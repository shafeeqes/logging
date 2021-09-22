// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package controller

import (
	"fmt"
	"time"

	client "github.com/gardener/logging/pkg/client"
	"github.com/gardener/logging/pkg/config"
	"github.com/gardener/logging/pkg/metrics"
	"github.com/gardener/logging/pkg/types"
	"github.com/prometheus/common/model"

	extensioncontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// GetClient search a client with <name> and returned if found.
// In case the controller is closed it returns true as second return value.
func (ctl *controller) GetClient(name string) (types.LokiClient, bool) {
	ctl.lock.RLocker().Lock()
	defer ctl.lock.RLocker().Unlock()

	if ctl.isStopped() {
		return nil, true
	}

	if client, ok := ctl.clients[name]; ok {
		return client, false
	}

	return nil, false
}

func (ctl *controller) newControllerClient(clientConf *config.Config) (ControllerClient, error) {
	mainClient, err := client.NewClient(clientConf, ctl.logger)
	if err != nil {
		return nil, err
	}
	c := &controllerClient{
		mainClient:        mainClient,
		defaultClient:     ctl.defaultClient,
		state:             clusterStateCreation,
		defaultClientConf: &ctl.conf.ControllerConfig.DefaultControllerClientConfig,
		mainClientConf:    &ctl.conf.ControllerConfig.MainControllerClientConfig,
		logger:            ctl.logger,
		name:              clientConf.ClientConfig.GrafanaLokiConfig.URL.Host,
	}

	c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInCreationState
	c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInCreationState

	return c, nil
}

func (ctl *controller) createControllerClient(cluster *extensionsv1alpha1.Cluster) {
	clientConf := ctl.getClientConfig(cluster.Name)
	if clientConf == nil {
		return
	}

	client, err := ctl.newControllerClient(clientConf)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorFailedToMakeLokiClient).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("failed to make new loki client for cluster %v", cluster.Name), "error", err.Error())
		return
	}

	ctl.updateControllerClientState(client, cluster)

	level.Info(ctl.logger).Log("msg", fmt.Sprintf("Add client for cluster %v in %v state", cluster.Name, client.GetState()))
	ctl.lock.Lock()
	defer ctl.lock.Unlock()

	if ctl.isStopped() {
		return
	}
	ctl.clients[cluster.Name] = client
}

func (ctl *controller) deleteControllerClient(cluster *extensionsv1alpha1.Cluster) {
	ctl.lock.Lock()

	if ctl.isStopped() {
		ctl.lock.Unlock()
		return
	}

	client, ok := ctl.clients[cluster.Name]
	if ok && client != nil {
		delete(ctl.clients, cluster.Name)
	}

	ctl.lock.Unlock()
	if ok && client != nil {
		level.Info(ctl.logger).Log("msg", fmt.Sprintf("Delete client for cluster %v", cluster.Name))
		client.StopWait()
	}
}

func (ctl *controller) updateControllerClientState(client ControllerClient, cluster *extensionsv1alpha1.Cluster) {

	shoot, err := extensioncontroller.ShootFromCluster(ctl.decoder, cluster)
	if err != nil {
		metrics.Errors.WithLabelValues(metrics.ErrorCanNotExtractShoot).Inc()
		level.Error(ctl.logger).Log("msg", fmt.Sprintf("can't extract shoot from cluster %v", cluster.Name))
	}

	client.SetState(getShootState(shoot))
}

// ClusterState is a type alias for string.
type clusterState string

const (
	clusterStateCreation    clusterState = "creation"
	clusterStateReady       clusterState = "ready"
	clusterStateHibernating clusterState = "hibernating"
	clusterStateHibernated  clusterState = "hibernated"
	clusterStateWakingUp    clusterState = "waking"
	clusterStateDeletion    clusterState = "deletion"
	clusterStateDeleted     clusterState = "deleted"
	clusterStateMigration   clusterState = "migration"
	clusterStateRestore     clusterState = "restore"
)

// Because loosing some logs when switching on and off client is not important we are omiting the synchronization.
type controllerClient struct {
	mainClient        types.LokiClient
	defaultClient     types.LokiClient
	muteMainClient    bool
	muteDefaultClient bool
	state             clusterState
	defaultClientConf *config.ControllerClientConfiguration
	mainClientConf    *config.ControllerClientConfiguration
	logger            log.Logger
	name              string
}

type ControllerClient interface {
	types.LokiClient
	GetState() clusterState
	SetState(state clusterState)
}

func (c *controllerClient) Handle(ls model.LabelSet, t time.Time, s string) error {
	if !c.muteMainClient {
		if err := c.mainClient.Handle(ls, t, s); err != nil {
			return err
		}
	}
	if !c.muteDefaultClient {
		return c.defaultClient.Handle(ls, t, s)
	}
	return nil
}

// Stop the client.
func (c *controllerClient) Stop() {
	c.mainClient.Stop()
}

// StopWait stops the client waiting all saved logs to be sent.
func (c *controllerClient) StopWait() {
	c.mainClient.StopWait()
}

func (c *controllerClient) SetState(state clusterState) {
	if state == c.state {
		return
	}

	switch state {
	case clusterStateReady:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInReadyState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInReadyState
	case clusterStateHibernating:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInHibernatingState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInHibernatingState
	case clusterStateWakingUp:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInWakingState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInWakingState
	case clusterStateDeletion:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInDeletionState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInDeletionState
	case clusterStateDeleted:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInDeletedState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInDeletedState
	case clusterStateHibernated:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInHibernatedState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInHibernatedState
	case clusterStateRestore:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInRestoreState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInRestoreState
	case clusterStateMigration:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInMigrationState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInMigrationState
	case clusterStateCreation:
		c.muteMainClient = !c.mainClientConf.SendLogsWhenIsInCreationState
		c.muteDefaultClient = !c.defaultClientConf.SendLogsWhenIsInCreationState
	default:
		level.Error(c.logger).Log("msg", fmt.Sprintf("Unknown state %v for cluster %v. The client state will not be changed", state, c.name))
		return
	}

	level.Info(c.logger).Log("msg", fmt.Sprintf("Cluster %s state changes from %v to %v", c.name, c.state, state))
	c.state = state
}

func (c *controllerClient) GetState() clusterState {
	return c.state
}
