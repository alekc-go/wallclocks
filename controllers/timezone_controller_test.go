package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/types"

	"alekc.dev/wallclocks/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("TimeZone controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 500
	)
	Context("When creating a TimeZone resource", func() {
		It("Should reconcile", func() {
			By("Creating a new TimeZone createdClock")
			ctx := context.Background()
			newTimeZone := &v1beta1.Timezone{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1beta1.GroupVersion.String(),
					Kind:       "TimeZone",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-timezone",
					Namespace: "test-namespace",
				},
				Spec: v1beta1.TimezoneSpec{
					Timezones: []string{
						"America/Los_Angeles",
						"Europe/London",
						"Europe/Moscow",
					},
				},
			}
			Expect(k8sClient.Create(ctx, newTimeZone)).Should(Succeed())

			createdTz := &v1beta1.Timezone{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: newTimeZone.ObjectMeta.Name, Namespace: newTimeZone.ObjectMeta.Namespace}, createdTz)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			//verify that 3 dependent objects have been created, and they are updating their values
			expectedTimeZones := map[string]string{
				"test-timezone-america-los-angeles": "America/Los_Angeles",
				"test-timezone-europe-london":       "Europe/London",
				"test-timezone-europe-moscow":       "Europe/Moscow",
			}
			for name, timezoneValue := range expectedTimeZones {
				createdClock := &v1beta1.WallClock{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: newTimeZone.ObjectMeta.Namespace}, createdClock)
					if err != nil {
						return false
					}
					//if the time was not updated yet, retry
					if createdClock.Status.Time == "" {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())
				Expect(metav1.GetControllerOf(createdClock).UID).To(BeEquivalentTo(newTimeZone.UID))
				Expect(createdClock.Spec.Timezone).To(BeEquivalentTo(timezoneValue))
				Expect(createdClock.Status.Time).NotTo(BeEmpty())

				//lets try to wait and validate that time has changed
				oldTime := createdClock.Status.Time
				time.Sleep(time.Millisecond * 1500)
				createdClock = &v1beta1.WallClock{}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: newTimeZone.ObjectMeta.Namespace}, createdClock)
					if err != nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())
				Expect(createdClock.Status.Time).NotTo(BeEquivalentTo(oldTime))
			}

			//remove one timezone from the object and verify that it has been deleted
			createdTz.Spec.Timezones = []string{
				"America/Los_Angeles",
				"Europe/London",
			}
			Expect(k8sClient.Update(ctx, createdTz)).Should(Succeed())

			createdTz = &v1beta1.Timezone{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: newTimeZone.ObjectMeta.Name, Namespace: newTimeZone.ObjectMeta.Namespace}, createdTz)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(len(createdTz.Spec.Timezones)).To(BeEquivalentTo(2))

			//verify that the clock is not there
			Eventually(func() bool {
				wallClock := &v1beta1.WallClock{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-timezone-europe-moscow", Namespace: newTimeZone.ObjectMeta.Namespace}, wallClock)
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())

			//Append a new clock
			createdTz.Spec.Timezones = []string{
				"America/Los_Angeles",
				"Europe/London",
				"Europe/Rome",
			}
			Expect(k8sClient.Update(ctx, createdTz)).Should(Succeed())

			//verify that clock exists
			wallClock := &v1beta1.WallClock{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-timezone-europe-rome", Namespace: newTimeZone.ObjectMeta.Namespace}, wallClock)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			//delete the object and validate that a new one has been created
			uid := wallClock.UID
			Expect(k8sClient.Delete(ctx, wallClock.DeepCopy())).Should(Succeed())

			//wait for deletion
			Eventually(func() bool {
				wallClock = &v1beta1.WallClock{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-timezone-europe-rome", Namespace: newTimeZone.ObjectMeta.Namespace}, wallClock)
				if err != nil || wallClock.UID == uid {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			wallClock = &v1beta1.WallClock{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-timezone-europe-rome", Namespace: newTimeZone.ObjectMeta.Namespace}, wallClock)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(wallClock.UID).NotTo(BeEquivalentTo(uid))
		})
	})
})
