package core

import "context"

// BackgroundService 用来把后台循环任务适配到统一的 Service 生命周期。
type BackgroundService struct {
	start  func(ctx context.Context)
	cancel context.CancelFunc
}

func NewBackgroundService(start func(ctx context.Context), cancel context.CancelFunc) *BackgroundService {
	return &BackgroundService{start: start, cancel: cancel}
}

func (s *BackgroundService) Start(ctx context.Context) error {
	s.start(ctx)
	return nil
}

func (s *BackgroundService) Stop(ctx context.Context) error {
	s.cancel()
	return nil
}
