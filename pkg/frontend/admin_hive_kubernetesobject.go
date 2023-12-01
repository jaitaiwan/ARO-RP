package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gobuffalo/flect"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

func (f *frontend) deleteAdminHiveKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	err := f._deleteAdminHiveKubernetesObjects(ctx, r, log)

	adminReply(log, w, nil, nil, err)
}

func (f *frontend) _deleteAdminHiveKubernetesObjects(ctx context.Context, r *http.Request, log *logrus.Entry) error {
	groupKind, name := chi.URLParam(r, "groupKind"), chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")
	force := strings.EqualFold(r.URL.Query().Get("force"), "true")

	if force {
		err := validateAdminKubernetesObjectsForceDelete(groupKind)
		if err != nil {
			return err
		}
	}

	// Retrieve the group kind
	gk := schema.ParseGroupKind(groupKind)

	// Delete ClusterDocument should be forbidden
	versions := scheme.Scheme.VersionsForGroupKind(gk)
	if len(versions) < 1 {
		return fmt.Errorf("failed retrieving versions for groupKind: %s", groupKind)
	}

	// ASSUMPTION: We use the first version of the resource here, but that may not be the right one
	return f.hiveClusterManager.DeleteHiveResource(ctx, gk.WithVersion(versions[0].Version), name, namespace)
}

func (f *frontend) getAdminHiveKubernetesObjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getAdminHiveKubernetesObjects(ctx, r, log)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminHiveKubernetesObjects(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	var err error

	groupKind, name := chi.URLParam(r, "groupKind"), chi.URLParam(r, "name")
	namespace := r.URL.Query().Get("namespace")

	gk := schema.ParseGroupKind(groupKind)

	versions := scheme.Scheme.VersionsForGroupKind(gk)
	if len(versions) < 1 {
		return nil, fmt.Errorf("failed retrieving versions for groupKind: %s", groupKind)
	}

	gvk := gk.WithVersion(versions[0].Version)
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: strings.ToLower(flect.Pluralize(gvk.Kind)),
	}

	// Cluster service principals are stored in secrets alongside cluster documents. This will prevent access.
	err = validateAdminKubernetesObjects(r.Method, gvr, namespace, name)
	if err != nil {
		return nil, err
	}

	if name != "" {
		un, err := f.hiveClusterManager.RetrieveHiveResource(ctx, gvk, name, namespace)
		if err != nil {
			return nil, err
		}
		return un.MarshalJSON()
	}
	return nil, fmt.Errorf("listing hive objects not supported")
}
