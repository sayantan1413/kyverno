package policymutation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	kyverno "github.com/kyverno/kyverno/pkg/api/kyverno/v1"
	"github.com/kyverno/kyverno/pkg/engine"
	"github.com/kyverno/kyverno/pkg/engine/variables"
	"github.com/kyverno/kyverno/pkg/utils"
)

func generateCronJobRule(rule kyverno.Rule, controllers string, log logr.Logger) kyvernoRule {
	logger := log.WithName("handleCronJob")

	hasCronJob := strings.Contains(controllers, engine.PodControllerCronJob) || strings.Contains(controllers, "all")
	if !hasCronJob {
		return kyvernoRule{}
	}

	logger.V(3).Info("generating rule for cronJob")
	jobRule := generateRuleForControllers(rule, "Job", logger)

	if reflect.DeepEqual(jobRule, kyvernoRule{}) {
		return kyvernoRule{}
	}

	cronJobRule := &jobRule

	name := fmt.Sprintf("autogen-cronjob-%s", rule.Name)
	if len(name) > 63 {
		name = name[:63]
	}
	cronJobRule.Name = name

	if len(jobRule.MatchResources.Any) > 0 {
		rule := cronJobAnyAllAutogenRule(cronJobRule.MatchResources.Any)
		cronJobRule.MatchResources.Any = rule
	} else if len(jobRule.MatchResources.All) > 0 {
		rule := cronJobAnyAllAutogenRule(cronJobRule.MatchResources.All)
		cronJobRule.MatchResources.All = rule
	} else {
		cronJobRule.MatchResources.Kinds = []string{engine.PodControllerCronJob}
	}

	if (jobRule.ExcludeResources) != nil && len(jobRule.ExcludeResources.Any) > 0 {
		rule := cronJobAnyAllAutogenRule(cronJobRule.ExcludeResources.Any)
		cronJobRule.ExcludeResources.Any = rule
	} else if (jobRule.ExcludeResources) != nil && len(jobRule.ExcludeResources.All) > 0 {
		rule := cronJobAnyAllAutogenRule(cronJobRule.ExcludeResources.All)
		cronJobRule.ExcludeResources.All = rule
	} else {
		if (jobRule.ExcludeResources) != nil && (len(jobRule.ExcludeResources.Kinds) > 0) {
			cronJobRule.ExcludeResources.Kinds = []string{engine.PodControllerCronJob}
		}
	}

	if (jobRule.Mutation != nil) && (jobRule.Mutation.Overlay != nil) {
		newMutation := &kyverno.Mutation{
			Overlay: map[string]interface{}{
				"spec": map[string]interface{}{
					"jobTemplate": jobRule.Mutation.Overlay,
				},
			},
		}

		cronJobRule.Mutation = newMutation.DeepCopy()
		return *cronJobRule
	}

	if (jobRule.Mutation != nil) && (jobRule.Mutation.PatchStrategicMerge != nil) {
		newMutation := &kyverno.Mutation{
			PatchStrategicMerge: map[string]interface{}{
				"spec": map[string]interface{}{
					"jobTemplate": jobRule.Mutation.PatchStrategicMerge,
				},
			},
		}
		cronJobRule.Mutation = newMutation.DeepCopy()
		return *cronJobRule
	}

	if (jobRule.Validation != nil) && (jobRule.Validation.Pattern != nil) {
		newValidate := &kyverno.Validation{
			Message: variables.FindAndShiftReferences(log, rule.Validation.Message, "spec/jobTemplate/spec/template", "pattern"),
			Pattern: map[string]interface{}{
				"spec": map[string]interface{}{
					"jobTemplate": jobRule.Validation.Pattern,
				},
			},
		}
		cronJobRule.Validation = newValidate.DeepCopy()
		return *cronJobRule
	}

	if (jobRule.Validation != nil) && (jobRule.Validation.ForEachValidation != nil) && (jobRule.Validation.ForEachValidation.Pattern != nil) {
		newValidate := &kyverno.Validation{
			Message:           variables.FindAndShiftReferences(log, rule.Validation.Message, "spec/jobTemplate/spec/template", "pattern"),
			ForEachValidation: jobRule.Validation.ForEachValidation,
		}
		cronJobRule.Validation = newValidate.DeepCopy()
		return *cronJobRule
	}

	if (jobRule.Validation != nil) && (jobRule.Validation.AnyPattern != nil) {
		var patterns []interface{}
		anyPatterns, err := jobRule.Validation.DeserializeAnyPattern()
		if err != nil {
			logger.Error(err, "failed to deserialize anyPattern, expect type array")
		}

		for _, pattern := range anyPatterns {
			newPattern := map[string]interface{}{
				"spec": map[string]interface{}{
					"jobTemplate": pattern,
				},
			}

			patterns = append(patterns, newPattern)
		}

		cronJobRule.Validation = &kyverno.Validation{
			Message:    variables.FindAndShiftReferences(log, rule.Validation.Message, "spec/jobTemplate/spec/template", "anyPattern"),
			AnyPattern: patterns,
		}
		return *cronJobRule
	}

	if (jobRule.Validation != nil) && (jobRule.Validation.ForEachValidation != nil) && (jobRule.Validation.ForEachValidation.AnyPattern != nil) {
		cronJobRule.Validation = &kyverno.Validation{
			Message:           variables.FindAndShiftReferences(log, rule.Validation.Message, "spec/jobTemplate/spec/template", "anyPattern"),
			ForEachValidation: jobRule.Validation.ForEachValidation,
		}
		return *cronJobRule
	}

	if (jobRule.Validation != nil) && (jobRule.Validation.ForEachValidation != nil) && (jobRule.Validation.ForEachValidation.Deny != nil) {
		cronJobRule.Validation = &kyverno.Validation{
			Message:           variables.FindAndShiftReferences(log, rule.Validation.Message, "spec/jobTemplate/spec/template", "anyPattern"),
			ForEachValidation: jobRule.Validation.ForEachValidation,
		}
		return *cronJobRule
	}

	if (jobRule.Mutation != nil) && (jobRule.Mutation.ForEachMutation != nil) && (jobRule.Mutation.ForEachMutation.PatchStrategicMerge != nil) {
		cronJobRule.Mutation = &kyverno.Mutation{
			ForEachMutation: jobRule.Mutation.ForEachMutation,
		}
		return *cronJobRule
	}

	if jobRule.VerifyImages != nil {
		newVerifyImages := make([]*kyverno.ImageVerification, len(jobRule.VerifyImages))
		for i, vi := range rule.VerifyImages {
			newVerifyImages[i] = vi.DeepCopy()
		}
		cronJobRule.VerifyImages = newVerifyImages
		return *cronJobRule
	}

	return kyvernoRule{}
}

// stripCronJob removes CronJob from controllers
func stripCronJob(controllers string) string {
	var newControllers []string

	controllerArr := strings.Split(controllers, ",")
	for _, c := range controllerArr {
		if c == engine.PodControllerCronJob {
			continue
		}

		newControllers = append(newControllers, c)
	}

	if len(newControllers) == 0 {
		return ""
	}

	return strings.Join(newControllers, ",")
}

func cronJobAnyAllAutogenRule(v kyverno.ResourceFilters) kyverno.ResourceFilters {
	anyKind := v.DeepCopy()
	for i, value := range v {
		if utils.ContainsPod(value.Kinds, "Job") {
			anyKind[i].Kinds = []string{engine.PodControllerCronJob}
		}
	}
	return anyKind
}
