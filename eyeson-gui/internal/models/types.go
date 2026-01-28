// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package models

type ResponseBase struct {
	Result  string `json:"result"`
	Message string `json:"message,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	ResponseBase
	UserId      int    `json:"userId,omitempty"`
	UserType    string `json:"userType,omitempty"`
	UserGroupId int    `json:"userGroupId,omitempty"`
	UserLevel   string `json:"userLevel,omitempty"`
	SessionId   string `json:"sessionId,omitempty"`
}

type GetProvisioningDataRequest struct {
	Username      string        `json:"username"`
	Password      string        `json:"password"`
	Start         int           `json:"start,omitempty"`
	Limit         int           `json:"limit,omitempty"`
	SortDirection string        `json:"sortDirection,omitempty"`
	Search        []SearchParam `json:"search,omitempty"`
}

type SearchParam struct {
	FieldName  string `json:"fieldName"`
	FieldValue string `json:"fieldValue"`
}

type SimData struct {
	CLI              string `json:"CLI"`
	MSISDN           string `json:"MSISDN"`
	SimStatusChange  string `json:"SIM_STATUS_CHANGE"`
	RatePlanFullName string `json:"RATE_PLAN_FULL_NAME"`
	CustomerLabel1   string `json:"CUSTOMER_LABEL_1"`
	CustomerLabel2   string `json:"CUSTOMER_LABEL_2"`
	CustomerLabel3   string `json:"CUSTOMER_LABEL_3"`
	SimSwap          string `json:"SIM_SWAP"`
	IMSI             string `json:"IMSI"`
	IMEI             string `json:"IMEI"`
	ApnName          string `json:"APN_NAME"`
	Ip1              string `json:"IP1"`
	MonthlyUsageMB   string `json:"MONTHLY_USAGE_MB"`
	AllocatedMB      string `json:"ALLOCATED_MB"`
	LastSessionTime  string `json:"LAST_SESSION_TIME"`
	InSession        string `json:"IN_SESSION"`
}

type GetProvisioningDataResponse struct {
	ResponseBase
	Count      int       `json:"count"`
	FieldNames []string  `json:"fieldNames"`
	Data       []SimData `json:"data"`
}

type ProvisioningAction struct {
	ActionType  string              `json:"actionType"`
	TargetValue string              `json:"targetValue"`
	TargetId    string              `json:"targetId,omitempty"`
	Subscribers []SubscriberRequest `json:"subscribers"`
}

type SubscriberRequest struct {
	NeId string `json:"neId"`
}

type UpdateProvisioningDataRequest struct {
	Username string               `json:"username"`
	Password string               `json:"password"`
	Actions  []ProvisioningAction `json:"actions"`
}

type UpdateProvisioningDataResponse struct {
	ResponseBase
	RequestId int `json:"requestId"`
}
