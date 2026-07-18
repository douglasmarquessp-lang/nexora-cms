package workflow

func (s *Service) ValidateJobStatus(status JobStatus) bool {
	switch status {
	case JobStatusDraft, JobStatusPending, JobStatusRunning,
		JobStatusPaused, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	}
	return false
}

func (s *Service) ValidateStepStatus(status StepStatus) bool {
	switch status {
	case StepStatusPending, StepStatusRunning, StepStatusCompleted,
		StepStatusFailed, StepStatusSkipped, StepStatusCancelled:
		return true
	}
	return false
}

func (s *Service) ValidateQueueStatus(status QueueStatus) bool {
	switch status {
	case QueueStatusPending, QueueStatusRunning, QueueStatusPaused,
		QueueStatusCompleted, QueueStatusFailed, QueueStatusCancelled:
		return true
	}
	return false
}

func (s *Service) ValidateWorkflowStep(step string) bool {
	for _, ws := range AllWorkflowSteps {
		if string(ws) == step {
			return true
		}
	}
	return false
}

func (s *Service) ValidatePriority(priority int) bool {
	return priority >= 1 && priority <= 10
}

func (s *Service) ValidateLanguage(lang string) bool {
	return lang == "pt" || lang == "en"
}

func (s *Service) ValidateAutomationAction(action string) bool {
	switch action {
	case "generate_article", "generate_pt_en", "publish_now",
		"schedule", "rebuild_seo", "regenerate", "duplicate":
		return true
	}
	return false
}
