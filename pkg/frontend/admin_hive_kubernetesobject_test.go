package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// TODO: Finish tests for the kubernetes object stuff

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	mock_adminactions "github.com/Azure/ARO-RP/pkg/util/mocks/adminactions"
	mock_hive "github.com/Azure/ARO-RP/pkg/util/mocks/hive"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestHiveAdminKubernetesObjectExistsGetAndDelete(t *testing.T) {
	podName := "generic"
	namespace := "test-namespace"

	ctx := context.Background()

	ti := newTestInfra(t).WithOpenShiftClusters().WithSubscriptions()
	defer ti.done()
	controller := gomock.NewController(t)
	defer controller.Finish()

	k := mock_adminactions.NewMockKubeActions(ti.controller)
	hiveClusterManager := mock_hive.NewMockClusterManager(controller)
	var ctxTyp = reflect.TypeOf((*context.Context)(nil)).Elem()
	var gvk = schema.GroupVersionKind{}

	// Get
	hiveClusterManager.EXPECT().RetrieveHiveResource(
		gomock.Any(), //  Context
		gomock.Any(), // schema.GroupVersionKind
		podName,      // string name
		namespace,    // string namespace
	).Return(&unstructured.Unstructured{}, nil).AnyTimes()

	// Delete
	hiveClusterManager.EXPECT().DeleteHiveResource(
		gomock.AssignableToTypeOf(ctxTyp), //  Context
		gomock.AssignableToTypeOf(gvk),    // schema.GroupVersionKind
		podName,                           // string name
		namespace,                         // string namespace
	).Return(nil).AnyTimes()

	// 404 Get
	hiveClusterManager.EXPECT().RetrieveHiveResource(
		gomock.AssignableToTypeOf(ctxTyp), //  Context
		gomock.AssignableToTypeOf(gvk),    // schema.GroupVersionKind
		podName,                           // string name
		namespace,                         // string namespace
	).Return(nil, errors.New("not found")).AnyTimes()

	// 404 Delete
	hiveClusterManager.EXPECT().DeleteHiveResource(
		gomock.AssignableToTypeOf(ctxTyp), //  Context
		gomock.AssignableToTypeOf(gvk),    // schema.GroupVersionKind
		podName,                           // string name
		namespace,                         // string namespace
	).Return(errors.New("some error")).AnyTimes()

	// Create new hive client set
	af := func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error) {
		return k, nil
	}

	// TODO: Setup the hive manager mock here
	// 4 cases to test here
	// exists get delete
	// 404 get delete

	f, err := NewFrontend(
		ctx,
		ti.audit, // Audit logger
		ti.log,   // General logger
		ti.env,
		ti.asyncOperationsDatabase,
		ti.clusterManagerDatabase,
		ti.openShiftClustersDatabase,
		ti.subscriptionsDatabase,
		nil, // Versions list
		api.APIs,
		&noop.Noop{},
		&noop.Noop{},
		nil,                // Aead encryption
		hiveClusterManager, // Hive cluster manager
		af,                 // Kube actions factory
		nil,                // Azure actions factory
		nil,                // "enricher"
	)
	if err != nil {
		t.Fatal(err)
	}

	go f.Run(ctx, nil, nil)

	// EXISTS GET
	requestStr := fmt.Sprintf("https://server/admin/hive/%s/%s&namespace=%s", "events", "abc", "test-namespace")

	resp, b, err := ti.request(http.MethodGet,
		requestStr,
		nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = validateResponse(resp, b, 200, "", []byte(`{}`+"\n"))
	if err != nil {
		t.Error(err)
	}

	// EXISTS DELETE
}
