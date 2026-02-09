// Copyright (c) 2026 Alexander G.
// Author: Alexander G. (Samsonix)
// License: MIT
// Project: EyesOn SIM Management System

package models

// Type aliases for reactive programming
type Sim = SimCard
type SimUpdate = SimData

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

type SearchParam struct {
	FieldName  string `json:"fieldName"`
	FieldValue string `json:"fieldValue"`
}

type GetProvisioningDataRequest struct {
	Username      string        `json:"username"`
	Password      string        `json:"password"`
	Start         int           `json:"start,omitempty"`
	Limit         int           `json:"limit,omitempty"`
	SortBy        string        `json:"sortBy,omitempty"`
	SortDirection string        `json:"sortDirection,omitempty"`
	Search        []SearchParam `json:"search,omitempty"`
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
	SyncStatus       string `json:"SYNC_STATUS,omitempty"` // PENDING, PROCESSING, or empty

	// Additional fields from Pelephone API
	EffectiveDate        string  `json:"EFFECTIVE_DATE"`
	ExpirationDate       string  `json:"EXPIRATION_DATE"`
	SimType              string  `json:"SIM_TYPE"`
	CustomerNumber       string  `json:"CUSTOMER_NUMBER"`
	CustomerName         string  `json:"CUSTOMER_NAME"`
	SubCustomerName      string  `json:"SUB_CUSTOMER_NAME"`
	OrderNumber          string  `json:"ORDER_NUMBER"`
	MonthlyUsageSMS      string  `json:"MONTHLY_USAGE_SMS"`
	BundleUtilization    string  `json:"BUNDLE_UTILIZATION"`
	PrepaidDataBalance   string  `json:"PREPAID_DATA_BALANCE"`
	DataThrottle         string  `json:"DATA_THROTTLE"`
	IsPooled             string  `json:"IS_POOLED"`
	RatePlanChange       string  `json:"RATE_PLAN_CHANGE"`
	RatePlanChangeRO     string  `json:"RATE_PLAN_CHANGE_READ_ONLY"`
	OneTimePackage       string  `json:"ONE_TIME_PACKAGE"`
	FutureSoc            *string `json:"FUTURE_SOC"`
	FutureSocName        *string `json:"FUTURE_SOC_NAME"`
	FutureEffectiveDate  *string `json:"FUTURE_EFFECTIVE_DATE"`
	FutureExpirationDate *string `json:"FUTURE_EXPIRATION_DATE"`
	ApnHlsfi             *string `json:"APNHLSFI"`
	ApnHname             *string `json:"APNHNAME"`
	SimRefresh           string  `json:"SIM_REFRESH"`
	RefreshSubUsages     string  `json:"REFRESH_SUBSCRIBER_USAGES"`
}

type GetProvisioningDataResponse struct {
	ResponseBase
	Count      int       `json:"count"`
	FieldNames []string  `json:"fieldNames"`
	Data       []SimData `json:"data"`
}

type SubscriberRequest struct {
	NeId string `json:"neId"`
}

type ProvisioningAction struct {
	ActionType  string              `json:"actionType"`
	TargetValue string              `json:"targetValue"`
	TargetId    string              `json:"targetId,omitempty"`
	Subscribers []SubscriberRequest `json:"subscribers"`
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

// BulkUpdateResponse is an alias for UpdateProvisioningDataResponse
type BulkUpdateResponse = UpdateProvisioningDataResponse

// Job models
type GetJobsRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Start     int    `json:"start,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	JobId     int    `json:"jobId,omitempty"`
	JobStatus string `json:"jobStatus,omitempty"`
}

type JobAction struct {
	NeId         string `json:"neId,omitempty"` // SIM identifier (CLI or MSISDN)
	RequestType  string `json:"requestType"`
	InitialValue string `json:"initialValue,omitempty"` // Value before change
	TargetValue  string `json:"targetValue"`
	Status       string `json:"status"`
	ErrorDesc    string `json:"errorDesc,omitempty"`
}

type Job struct {
	JobId          int         `json:"jobId"`
	JobStatus      string      `json:"jobStatus"`
	RequestTime    interface{} `json:"requestTime"`
	LastActionTime interface{} `json:"lastActionTime,omitempty"`
	Actions        []JobAction `json:"actions,omitempty"`
}

type GetJobsResponse struct {
	ResponseBase
	Count int   `json:"count"`
	Jobs  []Job `json:"jobs"`
}

// API Connection Status models
type APIConnectionInfo struct {
	Status         string            `json:"status"`
	ResponseTimeMs int               `json:"response_time_ms,omitempty"`
	Details        map[string]string `json:"details,omitempty"`
	Error          string            `json:"error,omitempty"`
}

type APIStatusResponse struct {
	EyesonAPI   APIConnectionInfo `json:"eyeson_api"`
	GoBackend   APIConnectionInfo `json:"go_backend"`
	Database    APIConnectionInfo `json:"database"`
	LastChecked string            `json:"last_checked"`
}

// Event represents a standard Server-Sent Event (SSE) structure
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
