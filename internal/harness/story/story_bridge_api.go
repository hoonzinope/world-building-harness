package story

import "context"

func OpenStore(root, packsRoot string) (*Store, error)    { return openStoryStore(root, packsRoot) }
func NewGMProvider(name string) GMProvider                { return newGMProvider(name) }
func (s *Store) ListStories() ([]Manifest, error)         { return s.listStories() }
func (s *Store) ReadManifest(id string) (Manifest, error) { return s.readManifest(id) }
func (s *Store) ReadState(id string) (State, error)       { return s.readState(id) }
func (s *Store) ReadTurns(id string) ([]Turn, error)      { return s.readTurns(id) }
func (s *Store) ReadQA(id string) ([]Question, error)     { return s.readQA(id) }
func (s *Store) CreateStory(m Manifest, st State, turns []Turn) error {
	return s.createStory(m, st, turns)
}
func (s *Store) WriteStoryTurnsProjection(id string, turns []Turn) error {
	return s.writeStoryTurnsProjection(id, turns)
}
func (s *Store) RewriteStorySummary(id, summary string) error {
	return s.rewriteStorySummary(id, summary)
}
func FindStoryTurn(turns []Turn, turnID int) (int, bool)           { return findStoryTurn(turns, turnID) }
func (s *Store) EnsureSeedStories(actorID string) error            { return s.ensureSeedStories(actorID) }
func (s *Store) ImportHector(actorID string) (string, bool, error) { return s.importHector(actorID) }
func (s *Store) RefreshHectorHistory(storyID, actorID string) error {
	return s.refreshHectorHistory(storyID, actorID)
}
func (s *Store) ParseHectorHistory() (HectorParsed, error) { return s.parseHectorHistory() }
func (s *Store) ReplaceStory(m Manifest, st State, turns []Turn) error {
	return s.replaceStory(m, st, turns)
}
func (s *Store) UpdateHectorImportedStory(m Manifest, parsed HectorParsed, hash string, imported bool) (string, bool, error) {
	return s.updateHectorImportedStory(m, parsed, hash, imported)
}
func (s *Store) AcquireLock(storyID, reason, actorID string) (func(), error) {
	return s.acquireLock(storyID, reason, actorID)
}
func (s *Store) ExportStoryBundle(storyID string, actorID, role string) (string, error) {
	return s.exportStoryBundle(storyID, &Actor{ID: actorID, Role: role})
}
func (s *Store) RecoverStory(storyID string) (RecoveryReport, error) { return s.recoverStory(storyID) }
func (s *Store) EditCurrentTurn(storyID, actorID, sceneBody, currentSituation string) error {
	return s.editCurrentTurn(storyID, actorID, sceneBody, currentSituation)
}
func (s *Store) RollbackStoryToTurn(storyID, actorID string, targetTurnID int) error {
	return s.rollbackStoryToTurn(storyID, actorID, targetTurnID)
}
func (s *Store) AppendChoice(storyID, actorID, role, choiceID, customMode, customText string) error {
	return s.appendChoice(storyID, &Actor{ID: actorID, Role: role}, choiceID, customMode, customText)
}
func (s *Store) AskQuestion(storyID, actorID, role, question string) error {
	return s.askQuestion(storyID, &Actor{ID: actorID, Role: role}, question)
}
func (s *Store) SubmitStoryInput(storyID, actorID, role string, currentTurnID int, idempotencyKey, choiceID, customMode, customText string) (string, error) {
	return s.submitStoryInput(storyID, &Actor{ID: actorID, Role: role}, currentTurnID, idempotencyKey, choiceID, customMode, customText)
}
func (s *Store) CreateStoryWithPrologueJob(actorID, title, style, characterName, traits string) (string, error) {
	return s.createStoryWithPrologueJob(actorID, title, style, characterName, traits)
}
func (s *Store) SubmitQuestionJob(storyID, actorID, role string, currentTurnID int, idempotencyKey, text string) (string, error) {
	return s.submitQuestionJob(storyID, &Actor{ID: actorID, Role: role}, currentTurnID, idempotencyKey, text)
}
func (s *Store) StartGMWorker(ctx context.Context, provider GMProvider) {
	s.startGMWorker(ctx, provider)
}
func (s *Store) ProcessOneGMJob(ctx context.Context, provider GMProvider) error {
	return s.processOneGMJob(ctx, provider)
}
func (s *Store) RunGMJob(ctx context.Context, provider GMProvider, job GMJob) error {
	return s.runGMJob(ctx, provider, job)
}
func (s *Store) MarkJobStarted(job GMJob) error { return s.markJobStarted(job) }
func (s *Store) ApplyGMOutput(job GMJob, out GMOutput, raw, providerName, model string) error {
	return s.applyGMOutput(job, out, raw, providerName, model)
}
func (s *Store) ApplyQuestionOutput(job GMJob, out GMOutput, raw, providerName, model string) error {
	return s.applyQuestionOutput(job, out, raw, providerName, model)
}
func (s *Store) FailGMJob(job GMJob, code, message string) error {
	return s.failGMJob(job, code, message)
}
func (s *Store) WriteFailedGMRaw(job GMJob, raw string) (string, error) {
	return s.writeFailedGMRaw(job, raw)
}
func (s *Store) BuildGMRequest(job GMJob) (GMRequest, error)    { return s.buildGMRequest(job) }
func (s *Store) ReadJob(storyID, jobID string) (GMJob, error)   { return s.readJob(storyID, jobID) }
func (s *Store) ListQueuedJobs(storyID string) ([]GMJob, error) { return s.listQueuedJobs(storyID) }
func (s *Store) ListJobs(storyID string) ([]GMJob, error)       { return s.listJobs(storyID) }
func (s *Store) ResumeFailedJob(storyID, actorID string) (string, error) {
	return s.resumeFailedJob(storyID, &Actor{ID: actorID})
}
func (s *Store) CancelFailedJob(storyID, actorID string) error {
	return s.cancelFailedJob(storyID, &Actor{ID: actorID})
}
func (s *Store) StoryHasBlockingGMJob(m Manifest) bool { return s.storyHasBlockingGMJob(m) }
func (s *Store) CreateDemoStory(actorID, title, style, characterName, traits string) (string, error) {
	return s.createDemoStory(actorID, title, style, characterName, traits)
}
func (s *Store) AppendStoryLifecycleEvent(storyID, actorID, eventType, fromStatus, toStatus string) error {
	return s.appendStoryLifecycleEvent(storyID, actorID, eventType, fromStatus, toStatus)
}
func (s *Store) ChangeStoryLifecycleStatus(storyID, actorID, nextStatus, eventType string) error {
	return s.changeStoryLifecycleStatus(storyID, actorID, nextStatus, eventType)
}
func (s *Store) ArchiveStory(storyID, actorID string) error { return s.archiveStory(storyID, actorID) }
func (s *Store) RestoreStory(storyID, actorID string) error { return s.restoreStory(storyID, actorID) }
func (s *Store) DeleteStory(storyID, actorID string) error  { return s.deleteStory(storyID, actorID) }
func (s *Store) AdminUpdateStory(storyID, actorID, status, activeDriver string) error {
	return s.adminUpdateStory(storyID, actorID, status, activeDriver)
}
func (s *Store) UpdateDriver(storyID, actorID, role, action string) error {
	return s.updateDriver(storyID, &Actor{ID: actorID, Role: role}, action)
}
