// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package services

import (
	"errors"
	"strings"

	"eyeson-go-server/internal/config"
	"eyeson-go-server/internal/database"
	"eyeson-go-server/internal/models"

	"gorm.io/gorm"
)

type UpstreamSelection string

const (
	UpstreamPelephone UpstreamSelection = "pelephone"
	UpstreamSimulator UpstreamSelection = "simulator"

	// DefaultPelephoneBaseURL is the hardcoded Pelephone API URL, used as source-of-truth
	// for manual sync regardless of EYESON_API_BASE_URL env setting.
	DefaultPelephoneBaseURL = "https://eot-portal.pelephone.co.il:8888"
)

const upstreamSelectedKey = "upstream.selected"

func NormalizeUpstreamSelection(v string) (UpstreamSelection, bool) {
	vv := strings.ToLower(strings.TrimSpace(v))
	switch vv {
	case string(UpstreamPelephone):
		return UpstreamPelephone, true
	case string(UpstreamSimulator):
		return UpstreamSimulator, true
	default:
		return UpstreamPelephone, false
	}
}

// GetUpstreamSelected reads the persisted upstream selection.
// If there is no setting yet, it defaults to pelephone.
func GetUpstreamSelected() (UpstreamSelection, error) {
	var setting models.SystemSetting
	err := database.DB.Where("key = ?", upstreamSelectedKey).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UpstreamPelephone, nil
		}
		return UpstreamPelephone, err
	}
	selected, ok := NormalizeUpstreamSelection(setting.Value)
	if !ok {
		return UpstreamPelephone, nil
	}
	return selected, nil
}

func SetUpstreamSelected(selected UpstreamSelection) error {
	// Validate
	if selected != UpstreamPelephone && selected != UpstreamSimulator {
		selected = UpstreamPelephone
	}
	setting := models.SystemSetting{Key: upstreamSelectedKey, Value: string(selected)}
	return database.DB.Save(&setting).Error
}

func ResolveUpstreamBaseURL(cfg *config.Config, selected UpstreamSelection) string {
	if cfg == nil {
		return ""
	}
	if selected == UpstreamSimulator {
		return strings.TrimSpace(cfg.SimulatorBaseUrl)
	}
	return strings.TrimSpace(cfg.ApiBaseUrl)
}
