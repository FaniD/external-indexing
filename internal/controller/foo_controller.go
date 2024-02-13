/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"strconv"

	mygroupv1alpha1 "github.com/FaniD/external-indexing/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	errCRNotFound      = errors.New("CR not found")
	errFailedToGetFoos = errors.New("failed to get foos")
	errFailedToGetCR   = errors.New("failed to get foo")
)

// FooReconciler reconciles a Foo object
type FooReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=mygroup.example.com,resources=foos,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mygroup.example.com,resources=foos/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mygroup.example.com,resources=foos/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Foo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *FooReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	foo, err := r.GetCR(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}

	l, err := r.GetListByPidUsingIndex(ctx, foo.Spec.Pid)
	if err != nil {
		return ctrl.Result{}, err
	}

	annotations := foo.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	foo.Annotations["count"] = strconv.Itoa(len(l))
	foo.SetAnnotations(annotations)

	err = r.Update(ctx, foo)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// Configure the informer with the indexer
func idxFunc(obj client.Object) []string {
	foo, ok := obj.(*mygroupv1alpha1.Foo)
	if !ok || foo.Spec.Pid == "" {
		return nil
	}
	return []string{foo.Spec.Pid}
}

// SetupWithManager sets up the controller with the Manager.
func (r *FooReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &mygroupv1alpha1.Foo{}, "pid", idxFunc); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&mygroupv1alpha1.Foo{}).
		Complete(r)
}

func (r *FooReconciler) GetCR(ctx context.Context, namespacedName types.NamespacedName) (*mygroupv1alpha1.Foo, error) {
	log := log.FromContext(ctx)
	foo := &mygroupv1alpha1.Foo{}
	err := r.Get(ctx, namespacedName, foo)

	if apierrors.IsNotFound(err) {
		log.Info("foo resource not found. Ignoring since object must be deleted")

		return nil, errCRNotFound
	}

	if err != nil {
		log.Error(err, "Failed to get foo")

		return nil, errFailedToGetCR
	}

	return foo, nil
}

func (r *FooReconciler) GetListByPid(ctx context.Context, pid string) ([]mygroupv1alpha1.Foo, error) {
	fooList := &mygroupv1alpha1.FooList{}

	if err := r.List(ctx, fooList); err != nil {
		return nil, errFailedToGetFoos
	}

	var filteredFooList []mygroupv1alpha1.Foo
	for _, foo := range fooList.Items {
		if foo.Spec.Pid == pid {
			filteredFooList = append(filteredFooList, foo)
		}
	}

	return filteredFooList, nil
}

func (r *FooReconciler) GetListByPidUsingIndex(ctx context.Context, pid string) ([]mygroupv1alpha1.Foo, error) {
	fooList := &mygroupv1alpha1.FooList{}

	if err := r.List(ctx, fooList, client.MatchingFields{"pid": pid}); err != nil {
		return nil, errFailedToGetFoos
	}

	return fooList.Items, nil
}
