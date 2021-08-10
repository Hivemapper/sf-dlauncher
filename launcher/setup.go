// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package launcher

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"syscall"

	"github.com/streamingfast/dgrpc"
	"github.com/streamingfast/dmetrics"
	"go.uber.org/zap"
)

func init() {
	dgrpc.Verbosity = 2
}

func SetupAnalyticsMetrics(metricsListenAddr string, pprofListenAddr string) {
	if metricsListenAddr != "" {
		go dmetrics.Serve(metricsListenAddr)
	}

	if err := SetMaxOpenFilesLimit(goodEnoughMaxOpenFilesLimit, osxStockMaxOpenFilesLimit); err != nil {
		UserLog.Warn("unable to adjust ulimit max open files value, it might causes problem along the road", zap.Error(err))
	}

	if pprofListenAddr != "" {
		go func() {
			err := http.ListenAndServe(pprofListenAddr, nil)
			if err != nil {
				UserLog.Debug("unable to start profiling server", zap.Error(err), zap.String("listen_addr", pprofListenAddr))
			}
		}()
	}
}

const goodEnoughMaxOpenFilesLimit uint64 = 1000000
const osxStockMaxOpenFilesLimit uint64 = 24576

func SetMaxOpenFilesLimit(goodEnoughMaxOpenFiles, osxStockMaxOpenFiles uint64) error {
	maxOpenFilesLimit, err := getMaxOpenFilesLimit()
	if err != nil {
		return err
	}

	UserLog.Debug("ulimit max open files before adjustment", zap.Uint64("current_value", maxOpenFilesLimit))
	if maxOpenFilesLimit >= goodEnoughMaxOpenFiles {
		UserLog.Debug("no need to update ulimit as it's already higher than our good enough value", zap.Uint64("good_enough_value", goodEnoughMaxOpenFiles))
		return nil
	}

	// We first try to set the value to our good enough value. It might or might not
	// work depending if the user permits the operation and if on OS X, the maximal
	// value possible as been increased (https://superuser.com/a/514049/459230).
	//
	// If our first try didn't work, let's try with a small value that should fit
	// most stock OS X value. This should probably be done only for OS X, other OSes
	// should probably even try a higher value than the minimal OS X value first.
	//
	// We might need conditional compilation units here to make the logic easier.
	err = trySetMaxOpenFilesLimit(goodEnoughMaxOpenFiles)
	if err != nil {
		UserLog.Debug("unable to use our good enough ulimit max open files value, going to try with something lower", zap.Error(err))
	} else {
		return logValueAfterAdjustment()
	}

	err = trySetMaxOpenFilesLimit(osxStockMaxOpenFiles)
	if err != nil {
		return fmt.Errorf("cannot set ulimit max open files: %w", err)
	}

	return logValueAfterAdjustment()
}

func trySetMaxOpenFilesLimit(value uint64) error {
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Cur: value,
		Max: value,
	})

	if err != nil {
		return fmt.Errorf("cannot set ulimit max open files: %w", err)
	}

	return nil
}

func getMaxOpenFilesLimit() (uint64, error) {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return 0, fmt.Errorf("cannot get ulimit max open files value: %w", err)
	}

	return rLimit.Cur, nil
}

func logValueAfterAdjustment() error {
	maxOpenFilesLimit, err := getMaxOpenFilesLimit()
	if err != nil {
		return err
	}

	UserLog.Debug("ulimit max open files after adjustment", zap.Uint64("current_value", maxOpenFilesLimit))
	return nil
}
