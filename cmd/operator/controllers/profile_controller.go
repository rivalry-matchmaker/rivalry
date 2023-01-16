/*
Copyright 2022.

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
	"encoding/json"
	"fmt"

	"github.com/rivalry-matchmaker/rivalry/cmd/operator/controllers/templates"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	zerolog "github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	omstreamv1alpha1 "github.com/rivalry-matchmaker/rivalry/cmd/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	FrontendName    = "rivalry-frontend"
	AccumulatorName = "rivalry-accumulator"
	DispenserName   = "rivalry-dispenser"
)

// ProfileReconciler reconciles a Profile object
type ProfileReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	profiles string
}

//+kubebuilder:rbac:groups=rivalry.rivalry,resources=profiles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rivalry.rivalry,resources=profiles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rivalry.rivalry,resources=profiles/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Profile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ProfileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	i := new(omstreamv1alpha1.Profile)
	if err := r.Client.Get(ctx, req.NamespacedName, i); err != nil {
		zerolog.Debug().Err(err).Str("req", req.String()).Msg("error getting profile")
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	zerolog.Debug().Str("req", req.String()).Interface("profile", i).Msg("profile fetched")

	r.profiles = profiles(i)

	if err := r.reconcileFrontend(ctx, i); err != nil {
		zerolog.Debug().Err(err).Str("req", req.String()).Msg("error reconciling the frontend")
		return ctrl.Result{}, err
	}

	if err := r.reconcileAccumulators(ctx, i); err != nil {
		zerolog.Debug().Err(err).Str("req", req.String()).Msg("error reconciling the accumulators")
		return ctrl.Result{}, err
	}

	if err := r.reconcileDispenser(ctx, i); err != nil {
		zerolog.Debug().Err(err).Str("req", req.String()).Msg("error reconciling the dispenser")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProfileReconciler) reconcileDispenser(ctx context.Context, i *omstreamv1alpha1.Profile) error {
	if dispenserIsConfigured, err := r.dispenserDeploymentIsConfigured(ctx, i); err != nil {
		zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
			Msg("error checking dispenser deployment is configured")
		return err
	} else {
		if !dispenserIsConfigured {
			if err := r.configureDispenserDeployment(ctx, i); err != nil {
				zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
					Msg("error configuring the dispenser deployment")
				return err
			}
		}
	}
	return nil
}

func (r *ProfileReconciler) dispenserDeploymentIsConfigured(ctx context.Context, i *omstreamv1alpha1.Profile) (bool, error) {
	s := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: i.Namespace, Name: DispenserName}, s); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ProfileReconciler) configureDispenserDeployment(ctx context.Context, i *omstreamv1alpha1.Profile) error {
	zerolog.Debug().Msg("configure dispenser called")
	s := new(appsv1.Deployment)
	if err := yaml.Unmarshal(templates.DispenserDeployment, s); err != nil {
		return err
	}

	s.ObjectMeta.Namespace = i.Namespace
	s.Spec.Template.Spec.Containers[0].Image = i.Spec.DispenserService.Image
	s.Spec.Template.Spec.Containers[0].Args = append(s.Spec.Template.Spec.Containers[0].Args,
		"--nats_addr", i.Spec.NatsAddress,
		"--redis_addr", i.Spec.RedisAddress,
		"--assignment_target", i.Spec.DispenserService.AssignmentTarget)

	if err := controllerutil.SetControllerReference(i, s, r.Scheme); err != nil {
		return err
	}

	if err := r.Client.Create(ctx, s); err != nil {
		return err
	}
	return nil
}

func (r *ProfileReconciler) reconcileAccumulators(ctx context.Context, i *omstreamv1alpha1.Profile) error {
	for _, matchProfile := range i.Spec.MatchProfiles {
		if accumulatorIsConfigured, err := r.accumulatorIsConfigured(ctx, i, matchProfile); err != nil {
			zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
				Msg("error checking accumulator deployment is configured")
			return err
		} else {
			if !accumulatorIsConfigured {
				if err := r.configureAccumulatorDeployment(ctx, i, matchProfile); err != nil {
					zerolog.Debug().Err(err).Str("match_profile", matchProfile.Name).
						Str("namespace", i.Namespace).Str("name", i.Name).
						Msg("error configuring the accumulator deployment")
					return err
				}
			}
		}
	}
	return nil
}

func accumulatorName(mp *omstreamv1alpha1.MatchProfile) string {
	return fmt.Sprintf("%s-%s", AccumulatorName, mp.Name)
}

func (r *ProfileReconciler) accumulatorIsConfigured(ctx context.Context, i *omstreamv1alpha1.Profile, mp *omstreamv1alpha1.MatchProfile) (bool, error) {
	s := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: i.Namespace, Name: accumulatorName(mp)}, s); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ProfileReconciler) configureAccumulatorDeployment(ctx context.Context, i *omstreamv1alpha1.Profile, mp *omstreamv1alpha1.MatchProfile) error {
	zerolog.Debug().Msg("configure accumulator called")
	s := new(appsv1.Deployment)
	if err := yaml.Unmarshal(templates.AccumulatorDeployment, s); err != nil {
		return err
	}

	s.ObjectMeta.Name = accumulatorName(mp)
	s.ObjectMeta.Namespace = i.Namespace
	s.Spec.Template.Spec.Containers[0].Image = i.Spec.AccumulatorService.Image
	s.Spec.Template.Spec.Containers[0].Args = append(s.Spec.Template.Spec.Containers[0].Args,
		"--nats_addr", i.Spec.NatsAddress,
		"--redis_addr", i.Spec.RedisAddress,
		"--profiles", r.profiles,
		"--profile", mp.Name,
		"--matchmaker_target", mp.MatchmakerTarget,
	)

	if err := controllerutil.SetControllerReference(i, s, r.Scheme); err != nil {
		return err
	}

	if err := r.Client.Create(ctx, s); err != nil {
		return err
	}
	return nil
}

func (r *ProfileReconciler) reconcileFrontend(ctx context.Context, i *omstreamv1alpha1.Profile) error {
	// check the frontend service is configured or configure it
	if frontendIsConfigured, err := r.frontendServiceIsConfigured(ctx, i); err != nil {
		zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
			Msg("error checking frontend service is configured")
		return err
	} else {
		if !frontendIsConfigured {
			if err := r.configureFrontendService(ctx, i); err != nil {
				zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
					Msg("error configuring the frontend service")
				return err
			}
		}
	}
	// check the frontend deployment is configured or configure it
	if frontendIsConfigured, err := r.frontendDeploymentIsConfigured(ctx, i); err != nil {
		zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
			Msg("error checking frontend deployment is configured")
		return err
	} else {
		if !frontendIsConfigured {
			if err := r.configureFrontendDeployment(ctx, i); err != nil {
				zerolog.Debug().Err(err).Str("namespace", i.Namespace).Str("name", i.Name).
					Msg("error configuring the frontend deployment")
				return err
			}
		}
	}
	return nil
}

func (r *ProfileReconciler) frontendDeploymentIsConfigured(ctx context.Context, i *omstreamv1alpha1.Profile) (bool, error) {
	s := new(appsv1.Deployment)
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: i.Namespace, Name: FrontendName}, s); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ProfileReconciler) configureFrontendDeployment(ctx context.Context, i *omstreamv1alpha1.Profile) error {
	zerolog.Debug().Msg("configure frontend called")
	s := new(appsv1.Deployment)
	if err := yaml.Unmarshal(templates.FrontendDeployment, s); err != nil {
		return err
	}

	s.ObjectMeta.Namespace = i.Namespace
	s.Spec.Template.Spec.Containers[0].Image = i.Spec.FrontendService.Image
	s.Spec.Template.Spec.Containers[0].Args = append(s.Spec.Template.Spec.Containers[0].Args,
		"--nats_addr", i.Spec.NatsAddress,
		"--redis_addr", i.Spec.RedisAddress,
		"--profiles", r.profiles)
	if len(i.Spec.FrontendService.ValidationTarget) != 0 {
		s.Spec.Template.Spec.Containers[0].Args = append(s.Spec.Template.Spec.Containers[0].Args,
			"--validation_target", i.Spec.FrontendService.ValidationTarget)
	}
	if len(i.Spec.FrontendService.DataTarget) != 0 {
		s.Spec.Template.Spec.Containers[0].Args = append(s.Spec.Template.Spec.Containers[0].Args,
			"--data_target", i.Spec.FrontendService.DataTarget)
	}

	if err := controllerutil.SetControllerReference(i, s, r.Scheme); err != nil {
		return err
	}

	if err := r.Client.Create(ctx, s); err != nil {
		return err
	}
	return nil
}

func profiles(profile *omstreamv1alpha1.Profile) string {
	matchProfiles := make([]*pb.MatchProfile, len(profile.Spec.MatchProfiles))
	for i, p := range profile.Spec.MatchProfiles {
		matchProfiles[i] = &pb.MatchProfile{Name: p.Name}
		for _, pool := range p.Pools {
			protoPool := &pb.Pool{Name: pool.Name}
			for _, sef := range pool.StringEqualsFilters {
				protoPool.StringEqualsFilters = append(protoPool.StringEqualsFilters,
					&pb.StringEqualsFilter{
						StringArg: sef.Arg,
						Value:     sef.Value,
					})
			}
			for _, drf := range pool.DoubleRangeFilters {
				protoPool.DoubleRangeFilters = append(protoPool.DoubleRangeFilters,
					&pb.DoubleRangeFilter{
						DoubleArg: drf.Arg,
						Min:       drf.Min,
						Max:       drf.Max,
					})
			}
			for _, tpf := range pool.TagPresentFilters {
				protoPool.TagPresentFilters = append(protoPool.TagPresentFilters,
					&pb.TagPresentFilter{
						Tag: tpf.Tag,
					})
			}
			matchProfiles[i].Pools = append(matchProfiles[i].Pools, protoPool)
		}
	}
	out, err := json.Marshal(matchProfiles)
	if err != nil {
		panic(err)
	}
	return string(out)
}

func (r *ProfileReconciler) frontendServiceIsConfigured(ctx context.Context, i *omstreamv1alpha1.Profile) (bool, error) {
	s := new(corev1.Service)
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: i.Namespace, Name: FrontendName}, s); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ProfileReconciler) configureFrontendService(ctx context.Context, i *omstreamv1alpha1.Profile) error {
	zerolog.Debug().Msg("configure frontend called")
	s := new(corev1.Service)
	if err := yaml.Unmarshal(templates.FrontendService, s); err != nil {
		return err
	}

	zerolog.Debug().Interface("service", s).Msg("service loaded from template")

	// set the namespace
	s.ObjectMeta.Namespace = i.Namespace

	if err := controllerutil.SetControllerReference(i, s, r.Scheme); err != nil {
		return err
	}

	if err := r.Client.Create(ctx, s); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProfileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&omstreamv1alpha1.Profile{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
