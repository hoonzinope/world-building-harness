package harness

import (
	"context"

	"github.com/hoonzi/world-harness/internal/harness/story"
)

type storyStore struct{ *story.Store }
type storyManifest = story.Manifest
type storyState = story.State
type storyChoice = story.Choice
type storyTurn = story.Turn
type storyQuestion = story.Question
type storyRecoveryReport = story.RecoveryReport
type gmJob = story.GMJob
type gmProvider = story.GMProvider
type gmOutput = story.GMOutput
type gmRequest = story.GMRequest
type gmOutputTurn = story.GMOutputTurn
type gmStatePatch = story.GMStatePatch

func openStoryStore(root, packsRoot string) (*storyStore, error) {
	s, err := story.OpenStore(root, packsRoot)
	if err != nil {
		return nil, err
	}
	return &storyStore{Store: s}, nil
}

func newGMProvider(name string) gmProvider { return story.NewGMProvider(name) }

func (s *storyStore) listStories() ([]storyManifest, error)         { return s.Store.ListStories() }
func (s *storyStore) readManifest(id string) (storyManifest, error) { return s.Store.ReadManifest(id) }
func (s *storyStore) readState(id string) (storyState, error)       { return s.Store.ReadState(id) }
func (s *storyStore) readTurns(id string) ([]storyTurn, error)      { return s.Store.ReadTurns(id) }
func (s *storyStore) readQA(id string) ([]storyQuestion, error)     { return s.Store.ReadQA(id) }
func (s *storyStore) createStory(m storyManifest, st storyState, turns []storyTurn) error {
	return s.Store.CreateStory(m, st, turns)
}
func (s *storyStore) writeStoryTurnsProjection(id string, turns []storyTurn) error {
	return s.Store.WriteStoryTurnsProjection(id, turns)
}
func (s *storyStore) rewriteStorySummary(id, summary string) error {
	return s.Store.RewriteStorySummary(id, summary)
}
func findStoryTurn(turns []storyTurn, turnID int) (int, bool) {
	return story.FindStoryTurn(turns, turnID)
}
func mustReadTurns(s *storyStore, storyID string) []storyTurn {
	turns, _ := s.readTurns(storyID)
	return turns
}
func (s *storyStore) ensureSeedStories(actorID string) error {
	return s.Store.EnsureSeedStories(actorID)
}
func (s *storyStore) importHector(actorID string) (string, bool, error) {
	return s.Store.ImportHector(actorID)
}
func (s *storyStore) refreshHectorHistory(storyID, actorID string) error {
	return s.Store.RefreshHectorHistory(storyID, actorID)
}
func (s *storyStore) parseHectorHistory() (story.HectorParsed, error) {
	return s.Store.ParseHectorHistory()
}
func (s *storyStore) replaceStory(m storyManifest, st storyState, turns []storyTurn) error {
	return s.Store.ReplaceStory(m, st, turns)
}
func (s *storyStore) updateHectorImportedStory(m storyManifest, parsed story.HectorParsed, hash string, imported bool) (string, bool, error) {
	return s.Store.UpdateHectorImportedStory(m, parsed, hash, imported)
}
func (s *storyStore) acquireLock(storyID, reason, actorID string) (func(), error) {
	return s.Store.AcquireLock(storyID, reason, actorID)
}
func (s *storyStore) exportStoryBundle(storyID string, actor *authUser) (string, error) {
	if actor == nil {
		return s.Store.ExportStoryBundle(storyID, "", "")
	}
	return s.Store.ExportStoryBundle(storyID, actor.ID, actor.Role)
}
func (s *storyStore) recoverStory(storyID string) (storyRecoveryReport, error) {
	return s.Store.RecoverStory(storyID)
}
func (s *storyStore) editCurrentTurn(storyID, actorID, sceneBody, currentSituation string) error {
	return s.Store.EditCurrentTurn(storyID, actorID, sceneBody, currentSituation)
}
func (s *storyStore) rollbackStoryToTurn(storyID, actorID string, targetTurnID int) error {
	return s.Store.RollbackStoryToTurn(storyID, actorID, targetTurnID)
}
func (s *storyStore) appendChoice(storyID string, u *authUser, choiceID, customMode, customText string) error {
	if u == nil {
		return s.Store.AppendChoice(storyID, "", "", choiceID, customMode, customText)
	}
	return s.Store.AppendChoice(storyID, u.ID, u.Role, choiceID, customMode, customText)
}
func (s *storyStore) askQuestion(storyID string, u *authUser, question string) error {
	if u == nil {
		return s.Store.AskQuestion(storyID, "", "", question)
	}
	return s.Store.AskQuestion(storyID, u.ID, u.Role, question)
}
func (s *storyStore) submitStoryInput(storyID string, u *authUser, currentTurnID int, idempotencyKey, choiceID, customMode, customText string) (string, error) {
	if u == nil {
		return s.Store.SubmitStoryInput(storyID, "", "", currentTurnID, idempotencyKey, choiceID, customMode, customText)
	}
	return s.Store.SubmitStoryInput(storyID, u.ID, u.Role, currentTurnID, idempotencyKey, choiceID, customMode, customText)
}
func (s *storyStore) createStoryWithPrologueJob(actorID, title, style, characterName, traits string) (string, error) {
	return s.Store.CreateStoryWithPrologueJob(actorID, title, style, characterName, traits)
}
func (s *storyStore) submitQuestionJob(storyID string, u *authUser, currentTurnID int, idempotencyKey, text string) (string, error) {
	if u == nil {
		return s.Store.SubmitQuestionJob(storyID, "", "", currentTurnID, idempotencyKey, text)
	}
	return s.Store.SubmitQuestionJob(storyID, u.ID, u.Role, currentTurnID, idempotencyKey, text)
}
func (s *storyStore) startGMWorker(ctx context.Context, provider gmProvider) {
	s.Store.StartGMWorker(ctx, provider)
}
func (s *storyStore) processOneGMJob(ctx context.Context, provider gmProvider) error {
	return s.Store.ProcessOneGMJob(ctx, provider)
}
func (s *storyStore) runGMJob(ctx context.Context, provider gmProvider, job gmJob) error {
	return s.Store.RunGMJob(ctx, provider, job)
}
func (s *storyStore) markJobStarted(job gmJob) error { return s.Store.MarkJobStarted(job) }
func (s *storyStore) applyGMOutput(job gmJob, out gmOutput, raw, providerName, model string) error {
	return s.Store.ApplyGMOutput(job, out, raw, providerName, model)
}
func (s *storyStore) applyQuestionOutput(job gmJob, out gmOutput, raw, providerName, model string) error {
	return s.Store.ApplyQuestionOutput(job, out, raw, providerName, model)
}
func (s *storyStore) failGMJob(job gmJob, code, message string) error {
	return s.Store.FailGMJob(job, code, message)
}
func (s *storyStore) writeFailedGMRaw(job gmJob, raw string) (string, error) {
	return s.Store.WriteFailedGMRaw(job, raw)
}
func (s *storyStore) buildGMRequest(job gmJob) (gmRequest, error) { return s.Store.BuildGMRequest(job) }
func (s *storyStore) readJob(storyID, jobID string) (gmJob, error) {
	return s.Store.ReadJob(storyID, jobID)
}
func (s *storyStore) listQueuedJobs(storyID string) ([]gmJob, error) {
	return s.Store.ListQueuedJobs(storyID)
}
func (s *storyStore) listJobs(storyID string) ([]gmJob, error) { return s.Store.ListJobs(storyID) }
func (s *storyStore) resumeFailedJob(storyID string, u *authUser) (string, error) {
	if u == nil {
		return s.Store.ResumeFailedJob(storyID, "")
	}
	return s.Store.ResumeFailedJob(storyID, u.ID)
}
func (s *storyStore) cancelFailedJob(storyID string, u *authUser) error {
	if u == nil {
		return s.Store.CancelFailedJob(storyID, "")
	}
	return s.Store.CancelFailedJob(storyID, u.ID)
}
func (s *storyStore) storyHasBlockingGMJob(m storyManifest) bool {
	return s.Store.StoryHasBlockingGMJob(m)
}
func (s *storyStore) createDemoStory(actorID, title, style, characterName, traits string) (string, error) {
	return s.Store.CreateDemoStory(actorID, title, style, characterName, traits)
}
func (s *storyStore) appendStoryLifecycleEvent(storyID, actorID, eventType, fromStatus, toStatus string) error {
	return s.Store.AppendStoryLifecycleEvent(storyID, actorID, eventType, fromStatus, toStatus)
}
func (s *storyStore) changeStoryLifecycleStatus(storyID, actorID, nextStatus, eventType string) error {
	return s.Store.ChangeStoryLifecycleStatus(storyID, actorID, nextStatus, eventType)
}
func (s *storyStore) archiveStory(storyID, actorID string) error {
	return s.Store.ArchiveStory(storyID, actorID)
}
func (s *storyStore) restoreStory(storyID, actorID string) error {
	return s.Store.RestoreStory(storyID, actorID)
}
func (s *storyStore) deleteStory(storyID, actorID string) error {
	return s.Store.DeleteStory(storyID, actorID)
}
func (s *storyStore) adminUpdateStory(storyID, actorID, status, activeDriver string) error {
	return s.Store.AdminUpdateStory(storyID, actorID, status, activeDriver)
}
func (s *storyStore) updateDriver(storyID string, u *authUser, action string) error {
	if u == nil {
		return s.Store.UpdateDriver(storyID, "", "", action)
	}
	return s.Store.UpdateDriver(storyID, u.ID, u.Role, action)
}
