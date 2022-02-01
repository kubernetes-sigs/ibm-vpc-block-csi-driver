/**
 * Copyright 2021 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package utils ...
package utils

import (
	"os"

	uid "github.com/gofrs/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/context"
)

// GetContextLogger ...
func GetContextLogger(ctx context.Context, isDebug bool) (*zap.Logger, string) {
	return GetContextLoggerWithRequestID(ctx, isDebug, nil)
}

// GetContextLoggerWithRequestID  adds existing requestID in the logger
// The Existing requestID might be coming from ControllerPublishVolume etc
func GetContextLoggerWithRequestID(ctx context.Context, isDebug bool, requestIDIn *string) (*zap.Logger, string) {
	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	traceLevel := zap.NewAtomicLevel()
	if isDebug {
		traceLevel.SetLevel(zap.DebugLevel)
	} else {
		traceLevel.SetLevel(zap.InfoLevel)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), consoleDebugging, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return (lvl >= traceLevel.Level()) && (lvl < zapcore.ErrorLevel)
		})),
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), consoleErrors, zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})),
	)
	logger := zap.New(core, zap.AddCaller())
	// generating a unique request ID so that logs can be filter
	if requestIDIn == nil {
		// Generate New RequestID if not provided
		uuid, _ := uid.NewV4() // #nosec G104: Attempt to randomly generate uuid
		requestID := uuid.String()
		requestIDIn = &requestID
	}
	logger = logger.With(zap.String("RequestID", *requestIDIn))
	return logger, *requestIDIn + " "
}
