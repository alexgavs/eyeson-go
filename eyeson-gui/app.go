package main

import (
	"context"
	"eyeson-gui/internal/api"
	"eyeson-gui/internal/models"
	"fmt"
)

type App struct {
	ctx    context.Context
	client *api.Client
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Login(username, password string) string {
	client, err := api.NewClient("https://eot-portal.pelephone.co.il:8888", username, password)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if err := client.Login(); err != nil {
		return fmt.Sprintf("Login failed: %v", err)
	}

	a.client = client
	return "SUCCESS"
}

func (a *App) GetSims(search string) ([]models.SimData, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not logged in")
	}

	var searchParams []models.SearchParam
	if search != "" {
		field := "MSISDN"
		if len(search) >= 3 && search[0:2] == "05" {
			field = "CLI"
		} else if len(search) >= 3 && search[0:3] == "972" {
			field = "MSISDN"
		}

		searchParams = append(searchParams, models.SearchParam{
			FieldName:  field,
			FieldValue: search,
		})
	}

	resp, err := a.client.GetSims(0, 50, searchParams)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (a *App) ChangeStatus(msisdn, status string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("not logged in")
	}

	resp, err := a.client.BulkUpdateStatus([]string{msisdn}, status)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Request ID: %d", resp.RequestId), nil
}
