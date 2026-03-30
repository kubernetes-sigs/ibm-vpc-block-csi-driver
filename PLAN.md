# Implementation Plan: mkfsOptions Parameter for IBM VPC Block CSI Driver

## Overview
Add capability to specify custom filesystem formatting options in StorageClass using a `mkfsOptions` parameter that will be passed to `mkfs` commands during volume formatting.

---

## 1. Design

### 1.1 StorageClass Parameter Structure

Add new parameter to StorageClass definitions:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ibmc-vpc-block-custom-mkfs
provisioner: vpc.block.csi.ibm.io
parameters:
  profile: "general-purpose"
  csi.storage.k8s.io/fstype: "ext4"
  mkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0 -O ^has_journal"
  # ... other parameters
```

**Parameter Format:**
- **Name:** `mkfsOptions`
- **Type:** String (space-separated options)
- **Optional:** Yes (defaults to driver's built-in options)
- **Scope:** Per-StorageClass
- **Propagation:** Controller → VolumeContext → Node

### 1.2 Data Flow

```
StorageClass (mkfsOptions)
    ↓
CreateVolume (extract from parameters)
    ↓
VolumeContext (store in labels)
    ↓
ControllerPublishVolume (pass through PublishContext)
    ↓
NodeStageVolume (extract from VolumeContext)
    ↓
FormatAndMountSensitiveWithFormatOptions (apply to mkfs)
    ↓
Filesystem Created with custom options
```

### 1.3 Current Implementation Analysis

**Formatting Location:** [`pkg/ibmcsidriver/node.go:298`](pkg/ibmcsidriver/node.go:298)

**Current Call:**
```go
err = csiNS.Mounter.GetSafeFormatAndMount().FormatAndMount(source, stagingTargetPath, fsType, options)
```

**Target Call (with formatOptions):**
```go
err = csiNS.Mounter.GetSafeFormatAndMount().FormatAndMountSensitiveWithFormatOptions(
    source, stagingTargetPath, fsType, options, nil, formatOptions)
```

**Underlying Implementation:** [`vendor/k8s.io/mount-utils/mount_linux.go:564-653`](vendor/k8s.io/mount-utils/mount_linux.go:564-653)

The `formatAndMountSensitive()` function:
1. Checks if disk is formatted using `blkid`
2. If unformatted, builds mkfs command with default options
3. Appends `formatOptions` to the command: `args = append(formatOptions, args...)`
4. Executes: `mkfs.<fstype> <formatOptions> <default-args> <source>`

---

## 2. Code Changes

### 2.1 Constants Definition

**File:** [`pkg/ibmcsidriver/constants.go`](pkg/ibmcsidriver/constants.go)

**Location:** After line 133 (after `Throughput` constant)

```go
const (
    // ... existing constants ...
    
    // Throughput ...
    Throughput = "throughput"
    
    // MkfsOptions - Custom mkfs options for filesystem formatting
    MkfsOptions = "mkfsOptions"
)
```

### 2.2 Controller Changes

**File:** [`pkg/ibmcsidriver/controller_helper.go`](pkg/ibmcsidriver/controller_helper.go)

**Location:** In `getVolumeParameters()` function, around line 142 in the switch statement

```go
func getVolumeParameters(logger *zap.Logger, req *csi.CreateVolumeRequest, driverConfig *config.Config) (*provider.Volume, error) {
    // ... existing code ...
    
    for key, value := range parameters {
        switch key {
        // ... existing cases ...
        
        case MkfsOptions:
            // Extract and validate mkfsOptions
            mkfsOpts := strings.TrimSpace(value)
            if mkfsOpts != "" {
                // Validate options for security
                if err := validateMkfsOptions(mkfsOpts, fsType); err != nil {
                    err = fmt.Errorf("invalid mkfsOptions: %v", err)
                    logger.Error("getVolumeParameters", zap.NamedError("InvalidParameter", err))
                    return nil, status.Error(codes.InvalidArgument, err.Error())
                }
                // Store in labels for VolumeContext
                labels[MkfsOptions] = mkfsOpts
                logger.Info("mkfsOptions specified", zap.String("mkfsOptions", mkfsOpts))
            }
        
        // ... rest of cases ...
        }
    }
    
    // ... rest of function ...
}
```

**Note:** The `labels` map is already returned as part of `VolumeContext` in the `createVolumeResponse()` function at line 448-450.

### 2.3 Node Service Changes

**File:** [`pkg/ibmcsidriver/node.go`](pkg/ibmcsidriver/node.go)

**Location:** In `NodeStageVolume()` function, replace lines 294-298

**Current code:**
```go
options := collectMountOptions(fsType, mnt.MountFlags)

// FormatAndMount will format only if needed
ctxLogger.Info("Formating and mounting ", zap.String("source", source), zap.String("stagingTargetPath", stagingTargetPath), zap.String("fsType", fsType), zap.Reflect("options", options))
err = csiNS.Mounter.GetSafeFormatAndMount().FormatAndMount(source, stagingTargetPath, fsType, options)
```

**New code:**
```go
options := collectMountOptions(fsType, mnt.MountFlags)

// Extract mkfsOptions from VolumeContext
volumeContext := req.GetVolumeContext()
formatOptions := extractFormatOptions(ctxLogger, volumeContext, fsType)

// FormatAndMount will format only if needed
ctxLogger.Info("Formatting and mounting",
    zap.String("source", source),
    zap.String("stagingTargetPath", stagingTargetPath),
    zap.String("fsType", fsType),
    zap.Reflect("mountOptions", options),
    zap.Reflect("formatOptions", formatOptions))

err = csiNS.Mounter.GetSafeFormatAndMount().FormatAndMountSensitiveWithFormatOptions(
    source, stagingTargetPath, fsType, options, nil, formatOptions)
```

**New Helper Function:** Add after `collectMountOptions()` at line 588

```go
func collectMountOptions(fsType string, mntFlags []string) []string {
    var options []string
    options = append(options, mntFlags...)

    // By default, xfs does not allow mounting of two volumes with the same filesystem uuid.
    // Force ignore this uuid to be able to mount volume + its clone / restored snapshot on the same node.
    if fsType == "xfs" {
        options = append(options, "nouuid")
    }
    return options
}

// extractFormatOptions extracts and validates mkfs options from VolumeContext
func extractFormatOptions(ctxLogger *zap.Logger, volumeContext map[string]string, fsType string) []string {
    mkfsOpts, exists := volumeContext[MkfsOptions]
    if !exists || mkfsOpts == "" {
        return nil
    }
    
    ctxLogger.Info("Using custom mkfs options", 
        zap.String("fsType", fsType),
        zap.String("mkfsOptions", mkfsOpts))
    
    // Parse space-separated options
    options := strings.Fields(mkfsOpts)
    
    // Validate again at node level for defense in depth
    if err := validateMkfsOptions(mkfsOpts, fsType); err != nil {
        ctxLogger.Warn("Invalid mkfs options detected at node level, ignoring",
            zap.String("mkfsOptions", mkfsOpts),
            zap.Error(err))
        return nil
    }
    
    return options
}
```

### 2.4 Validation Logic

**File:** [`pkg/ibmcsidriver/node_helper.go`](pkg/ibmcsidriver/node_helper.go)

**Location:** Add after `udevadmTrigger()` function (after line 167)

```go
// validateMkfsOptions validates mkfs options for security and correctness
func validateMkfsOptions(mkfsOpts string, fsType string) error {
    if mkfsOpts == "" {
        return nil
    }
    
    // Security: Block dangerous characters and patterns that could lead to command injection
    dangerousPatterns := []string{
        ";", "|", "&", "$", "`", "$(", "&&", "||",
        ">", "<", "\n", "\r", "\\", "'", "\"",
    }
    
    for _, pattern := range dangerousPatterns {
        if strings.Contains(mkfsOpts, pattern) {
            return fmt.Errorf("mkfsOptions contains forbidden character/pattern: %s", pattern)
        }
    }
    
    // Validate options are appropriate for filesystem type
    options := strings.Fields(mkfsOpts)
    for _, opt := range options {
        if !isValidMkfsOption(opt, fsType) {
            return fmt.Errorf("invalid mkfs option '%s' for filesystem type '%s'", opt, fsType)
        }
    }
    
    return nil
}

// isValidMkfsOption checks if an option is valid for the given filesystem type
func isValidMkfsOption(option string, fsType string) bool {
    // Must start with dash
    if !strings.HasPrefix(option, "-") {
        return false
    }
    
    // Filesystem-specific validation
    switch fsType {
    case FSTypeExt2, FSTypeExt3, FSTypeExt4:
        // Common ext* options: -b, -E, -F, -i, -I, -J, -m, -N, -O, -T
        // Allow both short form (-F) and long form with values (-E option=value)
        validPrefixes := []string{"-b", "-E", "-F", "-i", "-I", "-J", "-m", "-N", "-O", "-T"}
        for _, prefix := range validPrefixes {
            if strings.HasPrefix(option, prefix) {
                return true
            }
        }
    case FSTypeXfs:
        // Common xfs options: -b, -d, -f, -i, -l, -m, -n, -r, -s
        validPrefixes := []string{"-b", "-d", "-f", "-i", "-l", "-m", "-n", "-r", "-s"}
        for _, prefix := range validPrefixes {
            if strings.HasPrefix(option, prefix) {
                return true
            }
        }
    default:
        // For unknown filesystem types, be conservative
        return false
    }
    
    return false
}
```

---

## 3. Test Cases

### 3.1 Unit Tests

**File:** `pkg/ibmcsidriver/node_test.go`

```go
func TestExtractFormatOptions(t *testing.T) {
    logger, _ := zap.NewDevelopment()
    
    tests := []struct {
        name           string
        volumeContext  map[string]string
        fsType         string
        expectedOpts   []string
        expectNil      bool
    }{
        {
            name:          "No mkfsOptions",
            volumeContext: map[string]string{},
            fsType:        "ext4",
            expectNil:     true,
        },
        {
            name: "Valid ext4 options",
            volumeContext: map[string]string{
                MkfsOptions: "-E lazy_itable_init=0 -O ^has_journal",
            },
            fsType:       "ext4",
            expectedOpts: []string{"-E", "lazy_itable_init=0", "-O", "^has_journal"},
        },
        {
            name: "Valid xfs options",
            volumeContext: map[string]string{
                MkfsOptions: "-f -b size=4096",
            },
            fsType:       "xfs",
            expectedOpts: []string{"-f", "-b", "size=4096"},
        },
        {
            name: "Empty mkfsOptions",
            volumeContext: map[string]string{
                MkfsOptions: "",
            },
            fsType:    "ext4",
            expectNil: true,
        },
        {
            name: "Invalid options with dangerous characters",
            volumeContext: map[string]string{
                MkfsOptions: "-E test; rm -rf /",
            },
            fsType:    "ext4",
            expectNil: true, // Should be rejected and return nil
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := extractFormatOptions(logger, tt.volumeContext, tt.fsType)
            
            if tt.expectNil {
                if result != nil {
                    t.Errorf("Expected nil, got %v", result)
                }
            } else {
                if !reflect.DeepEqual(result, tt.expectedOpts) {
                    t.Errorf("Expected %v, got %v", tt.expectedOpts, result)
                }
            }
        })
    }
}

func TestValidateMkfsOptions(t *testing.T) {
    tests := []struct {
        name      string
        mkfsOpts  string
        fsType    string
        expectErr bool
    }{
        {
            name:      "Valid ext4 options",
            mkfsOpts:  "-E lazy_itable_init=0 -m0",
            fsType:    "ext4",
            expectErr: false,
        },
        {
            name:      "Valid xfs options",
            mkfsOpts:  "-f -b size=4096",
            fsType:    "xfs",
            expectErr: false,
        },
        {
            name:      "Command injection attempt with semicolon",
            mkfsOpts:  "-E test; rm -rf /",
            fsType:    "ext4",
            expectErr: true,
        },
        {
            name:      "Command injection attempt with pipe",
            mkfsOpts:  "-E test | cat /etc/passwd",
            fsType:    "ext4",
            expectErr: true,
        },
        {
            name:      "Invalid option for filesystem type",
            mkfsOpts:  "-Z invalid",
            fsType:    "ext4",
            expectErr: true,
        },
        {
            name:      "Empty options",
            mkfsOpts:  "",
            fsType:    "ext4",
            expectErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateMkfsOptions(tt.mkfsOpts, tt.fsType)
            
            if tt.expectErr && err == nil {
                t.Error("Expected error but got nil")
            }
            if !tt.expectErr && err != nil {
                t.Errorf("Expected no error but got: %v", err)
            }
        })
    }
}

func TestIsValidMkfsOption(t *testing.T) {
    tests := []struct {
        name     string
        option   string
        fsType   string
        expected bool
    }{
        {name: "ext4 -E option", option: "-E", fsType: "ext4", expected: true},
        {name: "ext4 -E with value", option: "-Elazy_itable_init=0", fsType: "ext4", expected: true},
        {name: "ext4 -m option", option: "-m0", fsType: "ext4", expected: true},
        {name: "xfs -f option", option: "-f", fsType: "xfs", expected: true},
        {name: "xfs -b option", option: "-b", fsType: "xfs", expected: true},
        {name: "invalid option no dash", option: "nodash", fsType: "ext4", expected: false},
        {name: "invalid option for ext4", option: "-Z", fsType: "ext4", expected: false},
        {name: "invalid option for xfs", option: "-Z", fsType: "xfs", expected: false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := isValidMkfsOption(tt.option, tt.fsType)
            if result != tt.expected {
                t.Errorf("Expected %v, got %v", tt.expected, result)
            }
        })
    }
}
```

### 3.2 Integration Tests

**File:** `pkg/ibmcsidriver/node_integration_test.go` (new file)

```go
func TestNodeStageVolumeWithMkfsOptions(t *testing.T) {
    // Test that mkfsOptions are properly extracted and passed through
    // This would require mocking the mounter and verifying the call
    
    tests := []struct {
        name          string
        volumeContext map[string]string
        fsType        string
        expectOptions []string
    }{
        {
            name: "ext4 with custom options",
            volumeContext: map[string]string{
                MkfsOptions: "-E lazy_itable_init=0 -m0",
            },
            fsType:        "ext4",
            expectOptions: []string{"-E", "lazy_itable_init=0", "-m0"},
        },
        {
            name: "xfs with custom options",
            volumeContext: map[string]string{
                MkfsOptions: "-f -b size=4096",
            },
            fsType:        "xfs",
            expectOptions: []string{"-f", "-b", "size=4096"},
        },
    }
    
    // Implementation would mock the mounter and verify FormatAndMountSensitiveWithFormatOptions
    // is called with the correct formatOptions parameter
}
```

### 3.3 End-to-End Tests

**Test Scenarios:**

1. **Basic Functionality Test**
   - Create StorageClass with mkfsOptions
   - Create PVC using the StorageClass
   - Verify volume is formatted with custom options
   - Verify volume mounts successfully

2. **Security Test**
   - Attempt to create StorageClass with malicious mkfsOptions
   - Verify validation rejects dangerous patterns
   - Verify volume creation fails gracefully

3. **Filesystem-Specific Tests**
   - Test ext4 with various valid options
   - Test xfs with various valid options
   - Test ext3 with various valid options

4. **Backward Compatibility Test**
   - Create StorageClass without mkfsOptions
   - Verify default behavior is unchanged
   - Verify existing volumes continue to work

---

## 4. Security Considerations

### 4.1 Command Injection Prevention

**Risk:** User-supplied mkfsOptions could contain malicious commands

**Mitigation:**
1. **Input Validation:** Strict whitelist of allowed characters and patterns
2. **Option Validation:** Only allow known-safe mkfs options for each filesystem type
3. **Defense in Depth:** Validate at both controller and node levels
4. **No Shell Execution:** Options are passed as array elements, not shell strings

**Blocked Patterns:**
- Shell metacharacters: `;`, `|`, `&`, `$`, `` ` ``, `>`, `<`
- Command substitution: `$(`, `` ` ``
- Quotes: `'`, `"`
- Newlines and control characters

### 4.2 Privilege Escalation Prevention

**Risk:** mkfs runs with elevated privileges

**Mitigation:**
1. Options are validated before execution
2. Only filesystem-specific options are allowed
3. No arbitrary commands can be injected
4. Kubernetes RBAC controls who can create StorageClasses

### 4.3 Audit and Logging

**Implementation:**
- Log all mkfsOptions at INFO level during volume creation
- Log validation failures at WARN level
- Include mkfsOptions in error messages for troubleshooting
- Ensure sensitive data is not logged

---

## 5. Documentation Updates

### 5.1 StorageClass Examples

**File:** `deploy/kubernetes/storageclass/ibmc-vpc-block-custom-mkfs-StorageClass.yaml` (new)

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ibmc-vpc-block-custom-mkfs
  annotations:
    description: "Custom StorageClass with mkfs options for optimized formatting"
provisioner: vpc.block.csi.ibm.io
parameters:
  profile: "general-purpose"
  csi.storage.k8s.io/fstype: "ext4"
  # Custom mkfs options for faster formatting (skip journal and inode table initialization)
  mkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0 -m0"
  billingType: "hourly"
  encrypted: "false"
  classVersion: "1"
reclaimPolicy: "Delete"
allowVolumeExpansion: true
```

### 5.2 README Updates

**File:** `README.md`

Add section on mkfsOptions parameter:

```markdown
## Advanced Configuration

### Custom Filesystem Formatting Options

You can specify custom mkfs options in your StorageClass to control how filesystems are formatted:

```yaml
parameters:
  csi.storage.k8s.io/fstype: "ext4"
  mkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0 -m0"
```

**Supported Options:**

- **ext4/ext3/ext2:** `-b`, `-E`, `-F`, `-i`, `-I`, `-J`, `-m`, `-N`, `-O`, `-T`
- **xfs:** `-b`, `-d`, `-f`, `-i`, `-l`, `-m`, `-n`, `-r`, `-s`

**Common Use Cases:**

1. **Faster formatting (ext4):**
   ```yaml
   mkfsOptions: "-E lazy_itable_init=0,lazy_journal_init=0"
   ```

2. **No reserved blocks (ext4):**
   ```yaml
   mkfsOptions: "-m0"
   ```

3. **Custom block size (xfs):**
   ```yaml
   mkfsOptions: "-b size=4096"
   ```

**Security Note:** Options are validated to prevent command injection. Only filesystem-specific options are allowed.
```

### 5.3 User Guide

**File:** `docs/mkfs-options-guide.md` (new)

Create comprehensive guide covering:
- Available options per filesystem type
- Common use cases and examples
- Performance implications
- Security considerations
- Troubleshooting

---

## 6. Implementation Checklist

- [ ] Add `MkfsOptions` constant to `constants.go`
- [ ] Implement `validateMkfsOptions()` in `node_helper.go`
- [ ] Implement `isValidMkfsOption()` in `node_helper.go`
- [ ] Add mkfsOptions extraction in `controller_helper.go`
- [ ] Implement `extractFormatOptions()` in `node.go`
- [ ] Update `NodeStageVolume()` to use `FormatAndMountSensitiveWithFormatOptions()`
- [ ] Write unit tests for validation functions
- [ ] Write unit tests for extraction functions
- [ ] Write integration tests
- [ ] Create example StorageClass with mkfsOptions
- [ ] Update README.md
- [ ] Create mkfs-options-guide.md
- [ ] Perform security review
- [ ] Test with various filesystem types
- [ ] Test backward compatibility
- [ ] Update CHANGELOG.md

---

## 7. Rollout Plan

### Phase 1: Development and Testing
1. Implement code changes
2. Write and run unit tests
3. Perform security review
4. Test in development environment

### Phase 2: Alpha Release
1. Release as alpha feature with documentation
2. Gather feedback from early adopters
3. Monitor for security issues
4. Refine validation logic based on feedback

### Phase 3: Beta Release
1. Address alpha feedback
2. Expand test coverage
3. Update documentation based on user feedback
4. Prepare for GA

### Phase 4: General Availability
1. Mark feature as stable
2. Include in release notes
3. Provide migration guide for existing users
4. Monitor adoption and issues

---

## 8. Risks and Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Command injection | High | Low | Strict input validation, whitelist approach |
| Breaking existing volumes | High | Low | Backward compatible, optional parameter |
| Performance impact | Medium | Low | Options are validated once, minimal overhead |
| User misconfiguration | Medium | Medium | Clear documentation, validation with helpful errors |
| Filesystem corruption | High | Very Low | Only allow safe, documented mkfs options |

---

## 9. Future Enhancements

1. **Preset Profiles:** Define common mkfsOptions profiles (fast, secure, balanced)
2. **Dynamic Validation:** Load allowed options from configuration
3. **Metrics:** Track usage of custom mkfsOptions
4. **Validation API:** Provide endpoint to validate mkfsOptions before creating StorageClass
5. **Extended Filesystem Support:** Add support for additional filesystem types (btrfs, f2fs)

---

## 10. References

- [Kubernetes CSI Specification](https://github.com/container-storage-interface/spec)
- [k8s.io/mount-utils Documentation](https://pkg.go.dev/k8s.io/mount-utils)
- [ext4 mkfs.ext4 Manual](https://man7.org/linux/man-pages/man8/mkfs.ext4.8.html)
- [XFS mkfs.xfs Manual](https://man7.org/linux/man-pages/man8/mkfs.xfs.8.html)
- [IBM VPC Block Storage Documentation](https://cloud.ibm.com/docs/vpc?topic=vpc-block-storage-about)