/*


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

package controllers

import (
	"context"
	"fmt"
	"time"
	"wallclocks/timezone"

	"k8s.io/utils/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	wallclocksv1beta1 "wallclocks/api/v1beta1"
)

// TimezoneReconciler reconciles a Timezone object
type TimezoneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=wallclocks.ziglu.io,resources=timezones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wallclocks.ziglu.io,resources=timezones/status,verbs=get;update;patch

func (r *TimezoneReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cLog := r.Log.WithValues("timezone", req.NamespacedName)

	//fetch the related timezone
	var tz = &wallclocksv1beta1.Timezone{}
	if err := r.Client.Get(ctx, req.NamespacedName, tz); err != nil {
		if errors.IsNotFound(err) {
			cLog.Error(err, "could not find requested tz")
			return reconcile.Result{Requeue: false}, nil
		}
		cLog.Error(err, "error while fetching required tz")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 15 * time.Second}, err
	}

	//loop through all requested timezones
	for _, location := range tz.Spec.Timezones {
		_, err := time.LoadLocation(location)
		// this can be done in a much cleaner way, for example using admission controller.
		// For the sake of this test, I am going simply to interrupt elaboration of this resource
		// And return an error
		if err != nil {
			cLog.Error(err, "the requested location is not valid", "location", location)
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 15 * time.Second}, err
		}
		//create a new WallClock
		_, err = createClock(ctx, r.Client, location, tz)
		if err != nil {
			return ctrl.Result{
				Requeue: false,
			}, err
		}
	}

	return ctrl.Result{
		Requeue: false,
	}, nil
}

func createClock(ctx context.Context, cl client.Client, location string, tz *wallclocksv1beta1.Timezone) (*wallclocksv1beta1.WallClock, error) {
	//get the clean location name
	cleanName := fmt.Sprintf("%s-%s", tz.Name, timezone.CleanName(location))

	//create wall clock
	clock := &wallclocksv1beta1.WallClock{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cleanName,
			Namespace: tz.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         tz.APIVersion,
				Kind:               tz.Kind,
				Name:               tz.Name,
				UID:                tz.UID,
				Controller:         pointer.BoolPtr(true),
				BlockOwnerDeletion: nil,
			}},
		},
		Spec:   wallclocksv1beta1.WallClockSpec{},
		Status: wallclocksv1beta1.WallClockStatus{},
	}
	err := cl.Create(ctx, clock)
	if err != nil {
		return nil, err
	}

	//retrieve the created wall clock and return it
	searchKey := client.ObjectKey{
		Namespace: tz.Namespace,
		Name:      cleanName,
	}
	if err := cl.Get(ctx, searchKey, clock); err != nil {
		return nil, err
	}
	return clock, nil
}

func (r *TimezoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wallclocksv1beta1.Timezone{}).
		Complete(r)
}
