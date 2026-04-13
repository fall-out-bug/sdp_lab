package model

import "time"

type Section struct {
	ID           string
	DocumentID   string
	Ordinal      int
	Heading      string
	CharStart    int
	CharEnd      int
	Preview      string
	Content      string
	ContentHash  string
	QualityFlags []string
	CreatedAt    time.Time
}
