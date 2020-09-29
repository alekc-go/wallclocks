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
	"sync"
	"time"
	"wallclocks/timezone"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/utils/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	Clocks sync.Map
}

const jobOwnerKey = ".metadata.controller"

// +kubebuilder:rbac:groups=wallclocks.ziglu.io,resources=timezones,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wallclocks.ziglu.io,resources=timezones/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wallclocks.ziglu.io,resources=wallclocks,verbs=get;list;watch;create;update;patch;delete

// StartClock starts the clock ticker and puts the closing channel into the schema
func (r *TimezoneReconciler) StartClock(clock wallclocksv1beta1.WallClock) {
	stopChannel := make(chan int)

	//load the location and validate it again
	loc, err := time.LoadLocation(clock.Spec.Timezone)
	if err != nil {
		r.Log.WithValues("wallClockName", clock.Name).Error(err, "wallclock has an invalid location stored")
		return
	}
	//persist our stop channel in our cache. We can use it later to stop the execution of the ticking in case of deleting the actual clock
	r.Clocks.Store(clock.UID, stopChannel)
	var patchValue string
	for {
		select {
		case <-stopChannel:
			close(stopChannel)
			r.Log.Info("stopping clock", "clockName", clock.Name)
			return
		default:
			//update our clock with newer value and persist
			patchValue = fmt.Sprintf(`{"status":{"time":"%s"}}`, time.Now().In(loc).Format("15:04:05"))

			//note, I am ignoring the error here simply for the lack of time to properly dealing with it.
			err = r.Client.Patch(context.Background(), &clock, client.RawPatch(types.MergePatchType, []byte(patchValue)))
			if err != nil {
				fmt.Println(err)
			}

			//sleep for a second, and then retry.
			//Note, if there is any delay linked to execution, it would mean that we will have a drift in time (so the end result
			//would be 0-1 sec
			//if it's essential that an object is being patched every second, then slightly different strategy is needed
			//perhaps spinning an additional goroutine
			time.Sleep(time.Second * 1)
		}
	}
}
func (r *TimezoneReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	cLog := r.Log.WithValues("timezone", req.NamespacedName)

	cLog.Info("Reconciling")

	//fetch the related timezone
	var tz = &wallclocksv1beta1.Timezone{}
	if err := r.Client.Get(ctx, req.NamespacedName, tz); err != nil {
		cLog.Error(err, "could not find requested tz")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//fetch child clocks (if we have any)
	clockMap, err := loadChildrenClocks(ctx, r.Client, tz)
	if err != nil {
		cLog.Error(err, "could not fetch children")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//loop through all requested timezones
	for _, location := range tz.Spec.Timezones {
		//if we have already this location for this specific timezone object, then skip the creation stage
		if clock, found := clockMap[location]; found {
			//remove the child from the clockMap
			//this would give some errors in case we have duplicated location in one timezone
			//but again, this kind of errors should be solved by an admission controller
			delete(clockMap, location)
			//only if this clock was not started yet, spawn the ticker
			if _, found = r.Clocks.Load(clock.UID); !found {
				go r.StartClock(clock)
			}
			continue
		}

		//verify if the location is a valid one
		_, err := time.LoadLocation(location)
		// this can be done in a much cleaner way, for example using admission controller.
		// For the sake of this test, I am going simply to interrupt elaboration of this resource
		// And return an error
		if err != nil {
			cLog.Error(err, "the requested location is not valid", "location", location)
			return reconcile.Result{}, err
		}
		//create a new WallClock
		_, err = createClock(ctx, r.Client, location, tz)
		if err != nil {
			return ctrl.Result{}, err
		}

		//we got our new clock. Lets persist the association
		//note, potentially we could have queried etcd for the list of child resources, but for the sake of
		//this exercise I prefer to keep explicit association
		//patch := client.MergeFrom(tz.DeepCopy())
		//err = r.Client.Patch(ctx, tz, patch)
		if err != nil {
			return ctrl.Result{
				Requeue: false,
			}, err
		}
	}

	//check if we have any orphans. If that's the case, stop the ticker, and delete the child
	for location, clock := range clockMap {
		//stop the ticker (if we have one up and running)
		if stopChannel, found := r.Clocks.LoadAndDelete(location); found {
			stopChannel.(chan int) <- 1
		}
		//for the sake of exercise I am going to ignore this atm
		_ = r.Delete(ctx, &clock)
	}

	return ctrl.Result{}, nil
}

// loadChildrenClocks loads all children from the parent tz
func loadChildrenClocks(ctx context.Context, cl client.Client, tz *wallclocksv1beta1.Timezone) (map[string]wallclocksv1beta1.WallClock, error) {
	//load children (if any) from the cluster
	result := make(map[string]wallclocksv1beta1.WallClock, 0)
	var clocks wallclocksv1beta1.WallClockList
	err := cl.List(ctx, &clocks, client.InNamespace(tz.Namespace), client.MatchingFields{jobOwnerKey: string(tz.UID)})
	if err != nil {
		return result, err
	}
	//organize results in a quick map
	for _, clock := range clocks.Items {
		result[clock.Spec.Timezone] = clock
	}
	return result, nil
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
		Spec: wallclocksv1beta1.WallClockSpec{
			Timezone: location,
		},
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
	if err := mgr.GetFieldIndexer().IndexField(&wallclocksv1beta1.WallClock{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		wallClock := rawObj.(*wallclocksv1beta1.WallClock)
		owner := metav1.GetControllerOf(wallClock)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != wallclocksv1beta1.GroupVersion.String() || owner.Kind != "Timezone" {
			return nil
		}
		return []string{string(owner.UID)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&wallclocksv1beta1.Timezone{}).
		Owns(&wallclocksv1beta1.WallClock{}).
		Complete(r)
}
