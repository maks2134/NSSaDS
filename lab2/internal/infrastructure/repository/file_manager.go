package repository

import (
	"NSSaDS/lab2/internal/domain"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileManager struct {
	uploadDir     string
	sessions      map[string]*domain.TransferSession
	sessionsMutex sync.RWMutex
	cleanupTicker *time.Ticker
}

func NewFileManager(uploadDir string) *FileManager {
	fm := &FileManager{
		uploadDir: uploadDir,
		sessions:  make(map[string]*domain.TransferSession),
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Warning: failed to create upload directory: %v\n", err)
	}

	fm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go fm.cleanupRoutine()

	return fm
}

func (fm *FileManager) SaveFile(filename string, data []byte, offset int64) error {
	filePath := filepath.Join(fm.uploadDir, filename)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek to offset: %w", err)
		}
	}

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func (fm *FileManager) ReadFile(filename string) ([]byte, error) {
	filePath := filepath.Join(fm.uploadDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

func (fm *FileManager) GetFileInfo(filename string) (*domain.FileInfo, error) {
	filePath := filepath.Join(fm.uploadDir, filename)

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &domain.FileInfo{
		Name:    filename,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		Path:    filePath,
	}, nil
}

func (fm *FileManager) DeleteFile(filename string) error {
	filePath := filepath.Join(fm.uploadDir, filename)

	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (fm *FileManager) CreateTransferSession(session *domain.TransferSession) error {
	fm.sessionsMutex.Lock()
	defer fm.sessionsMutex.Unlock()

	fm.sessions[session.ID] = session
	return nil
}

func (fm *FileManager) GetTransferSession(clientAddr string, filename string) (*domain.TransferSession, error) {
	fm.sessionsMutex.RLock()
	defer fm.sessionsMutex.RUnlock()

	for _, session := range fm.sessions {
		if session.ClientAddr == clientAddr && session.FileName == filename {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

func (fm *FileManager) UpdateTransferSession(session *domain.TransferSession) error {
	fm.sessionsMutex.Lock()
	defer fm.sessionsMutex.Unlock()

	if _, exists := fm.sessions[session.ID]; exists {
		fm.sessions[session.ID] = session
		return nil
	}

	return fmt.Errorf("session not found")
}

func (fm *FileManager) DeleteTransferSession(sessionID string) error {
	fm.sessionsMutex.Lock()
	defer fm.sessionsMutex.Unlock()

	delete(fm.sessions, sessionID)
	return nil
}

func (fm *FileManager) CleanupExpiredSessions() error {
	fm.sessionsMutex.Lock()
	defer fm.sessionsMutex.Unlock()

	now := time.Now()
	expiredSessions := []string{}

	for id, session := range fm.sessions {
		if now.Sub(session.LastUpdate) > 5*time.Minute {
			expiredSessions = append(expiredSessions, id)
		}
	}

	for _, id := range expiredSessions {
		delete(fm.sessions, id)
	}

	return nil
}

func (fm *FileManager) cleanupRoutine() {
	for range fm.cleanupTicker.C {
		if err := fm.CleanupExpiredSessions(); err != nil {
			fmt.Printf("Warning: failed to cleanup expired sessions: %v\n", err)
		}
	}
}

func (fm *FileManager) Close() {
	if fm.cleanupTicker != nil {
		fm.cleanupTicker.Stop()
	}
}
