package models

import "errors"

var (
	ErrOnlineConfKeyNotFound      = errors.New("onlineconf key not found")
	ErrOnlineConfKeyExists        = errors.New("onlineconf key already exists")
	ErrOnlineConfVersionConflict  = errors.New("onlineconf key version conflict")
	ErrOnlineConfSnapshotNotReady = errors.New("onlineconf snapshot is not loaded")
)
