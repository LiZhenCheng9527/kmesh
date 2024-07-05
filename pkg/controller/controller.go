/*
 * Copyright 2023 The Kmesh Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"context"
	"fmt"

	"kmesh.net/kmesh/daemon/options"
	"kmesh.net/kmesh/pkg/bpf"
	"kmesh.net/kmesh/pkg/constants"
	"kmesh.net/kmesh/pkg/controller/bypass"
	"kmesh.net/kmesh/pkg/controller/security"
	"kmesh.net/kmesh/pkg/logger"
	"kmesh.net/kmesh/pkg/utils"
)

var (
	stopCh      = make(chan struct{})
	ctx, cancle = context.WithCancel(context.Background())
	log         = logger.NewLoggerField("controller")
)

type Controller struct {
	mode                string
	bpfWorkloadObj      *bpf.BpfKmeshWorkload
	client              *XdsClient
	enableByPass        bool
	enableSecretManager bool
	bpfFsPath           string
	enableBpfLog        bool
}

func NewController(opts *options.BootstrapConfigs, bpfWorkloadObj *bpf.BpfKmeshWorkload, bpfFsPath string, enableBpfLog bool) *Controller {
	return &Controller{
		mode:                opts.BpfConfig.Mode,
		enableByPass:        opts.ByPassConfig.EnableByPass,
		bpfWorkloadObj:      bpfWorkloadObj,
		enableSecretManager: opts.SecretManagerConfig.Enable,
		bpfFsPath:           bpfFsPath,
		enableBpfLog:        enableBpfLog,
	}
}

func (c *Controller) Start() error {
	if c.enableByPass {
		clientset, err := utils.GetK8sclient()
		if err != nil {
			return err
		}

		err = bypass.StartByPassController(clientset)
		if err != nil {
			return fmt.Errorf("failed to start bypass controller: %v", err)
		}

		log.Info("start bypass controller successfully")
	}

	if c.mode != constants.WorkloadMode && c.mode != constants.AdsMode {
		return nil
	}

	if c.enableBpfLog {
		if err := logger.StartRingBufReader(ctx, c.mode, c.bpfFsPath); err != nil {
			return fmt.Errorf("fail to start ringbuf reader: %v", err)
		}
	}
	c.client = NewXdsClient(c.mode, c.bpfWorkloadObj)
	if c.client.WorkloadController != nil {
		if c.enableSecretManager {
			secertManager, err := security.NewSecretManager()
			if err != nil {
				return fmt.Errorf("secretManager create failed: %v", err)
			}
			go secertManager.Run(stopCh)
			c.client.WorkloadController.Processor.SecretManager = secertManager
		}
	}

	go c.client.WorkloadController.Run(ctx)

	return c.client.Run(stopCh)
}

func (c *Controller) Stop() {
	if c == nil {
		return
	}
	close(stopCh)
	cancle()
	if c.client != nil {
		c.client.Close()
	}
}

func (c *Controller) GetXdsClient() *XdsClient {
	return c.client
}
