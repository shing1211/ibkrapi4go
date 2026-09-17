// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/shing1211/ibkrapi4go/client"
	"github.com/shing1211/ibkrapi4go/internal"
)

// List returns all accounts associated with the client ID.
func (m *RESTAccounts) List(ctx context.Context) ([]RESTAccountSummary, error) {
	const op = "Accounts.List"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListAccountsWithResponse(ctx, nil)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw []RESTAccountSummary
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return raw, nil
}

// LoginMessages returns all login messages for the client ID.
func (m *RESTAccounts) LoginMessages(ctx context.Context) ([]LoginMessage, error) {
	const op = "Accounts.LoginMessages"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListAccountsLoginMessagesWithResponse(ctx, nil)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw LoginMessagesWrapper
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	out := make([]LoginMessage, 0, len(raw.LoginMessages))
	for _, lm := range raw.LoginMessages {
		if lm != nil {
			out = append(out, *lm)
		}
	}
	return out, nil
}

// BulkStatus returns status for all accounts associated with the client ID.
func (m *RESTAccounts) BulkStatus(ctx context.Context) ([]RESTAccountStatus, error) {
	const op = "Accounts.BulkStatus"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.ListAccountsStatusWithResponse(ctx, nil)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw AccountStatusBulkWrapper
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	out := make([]RESTAccountStatus, 0, len(raw.Accounts))
	for _, a := range raw.Accounts {
		if a != nil {
			out = append(out, *a)
		}
	}
	return out, nil
}

// KycURL returns the Au10Tix KYC URL for the given account.
func (m *RESTAccounts) KycURL(ctx context.Context, accountID AccountID) (string, error) {
	const op = "Accounts.KycURL"
	if err := m.surface.owner.checkOpen(); err != nil {
		return "", err
	}
	resp, err := m.surface.generated.GetAccountsKycWithResponse(ctx, string(accountID))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	var raw Au10TixWrapper
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return "", e
	}
	return raw.KycURL(), nil
}

// LoginMessagesForAccount returns login messages for a specific account.
func (m *RESTAccounts) LoginMessagesForAccount(ctx context.Context, accountID AccountID) ([]LoginMessage, error) {
	const op = "Accounts.LoginMessagesForAccount"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetAccountsLoginMessagesWithResponse(ctx, string(accountID), nil)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw LoginMessagesWrapper
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	out := make([]LoginMessage, 0, len(raw.LoginMessages))
	for _, lm := range raw.LoginMessages {
		if lm != nil {
			out = append(out, *lm)
		}
	}
	return out, nil
}

// Status returns the account status for a specific account.
func (m *RESTAccounts) Status(ctx context.Context, accountID AccountID) (*RESTAccountStatus, error) {
	const op = "Accounts.Status"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	resp, err := m.surface.generated.GetAccountsStatusWithResponse(ctx, string(accountID))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw RESTAccountStatus
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	return &raw, nil
}

// Tasks returns registration tasks for the given account.
func (m *RESTAccounts) Tasks(ctx context.Context, accountID AccountID, taskType string) ([]RegistrationTaskItem, error) {
	const op = "Accounts.Tasks"
	if err := m.surface.owner.checkOpen(); err != nil {
		return nil, err
	}
	params := client.GetAccountsTasksParams{}
	if taskType != "" {
		t := client.GetAccountsTasksParamsType(taskType)
		params.Type = &t
	}
	resp, err := m.surface.generated.GetAccountsTasksWithResponse(ctx, string(accountID), &params)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	var raw RegistrationTasksWrapper
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		e := &Error{Op: op, Message: "decode: " + err.Error(), Err: err}
		internal.LogError(m.surface.owner.cfg.logger, e)
		return nil, e
	}
	tasks := raw.Tasks()
	return tasks, nil
}

// Update updates the account with the given configuration.
func (m *RESTAccounts) Update(ctx context.Context, config AccountConfiguration) error {
	const op = "Accounts.Update"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	body := client.AccountConfiguration{
		AccountId: nil,
		Type:      config.Type,
		Value:     config.Value,
	}
	if config.AccountID != "" {
		s := string(config.AccountID)
		body.AccountId = &s
	}
	data, _ := json.Marshal(body)
	resp, err := m.surface.generated.UpdateAccountsWithBodyWithResponse(ctx, "application/json", strings.NewReader(string(data)))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// Create submits a new account application. The payload should be JSON matching
// the account creation schema; mimeType defaults to "application/json".
func (m *RESTAccounts) Create(ctx context.Context, payload io.Reader, mimeType string) error {
	const op = "Accounts.Create"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	if mimeType == "" {
		mimeType = "application/json"
	}
	data, err := io.ReadAll(payload)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	resp, err := m.surface.generated.CreateAccountsWithBodyWithResponse(ctx, mimeType, bytes.NewReader(data))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// SubmitDocument uploads a document (PDF) for the account.
func (m *RESTAccounts) SubmitDocument(ctx context.Context, accountID AccountID, doc io.Reader, filename, mimeType string) error {
	const op = "Accounts.SubmitDocument"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	if mimeType == "" {
		mimeType = "application/pdf"
	}
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if _, err := io.Copy(part, doc); err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if err := w.WriteField("accountId", string(accountID)); err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if err := w.Close(); err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	resp, err := m.surface.generated.CreateAccountsDocumentsWithBodyWithResponse(ctx, w.FormDataContentType(), buf)
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// UpdateStatus updates the status of an account.
func (m *RESTAccounts) UpdateStatus(ctx context.Context, accountID AccountID, status string) error {
	const op = "Accounts.UpdateStatus"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	body := client.AccountStatusRequest{Status: nil}
	if status != "" {
		s := client.AccountStatusRequestStatus(status)
		body.Status = &s
	}
	data, _ := json.Marshal(body)
	resp, err := m.surface.generated.UpdateAccountsStatusWithBodyWithResponse(ctx, string(accountID), "application/json", bytes.NewReader(data))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// UpdateTasks updates registration tasks for the account.
func (m *RESTAccounts) UpdateTasks(ctx context.Context, accountID AccountID, taskType string, updates []TaskUpdate) error {
	const op = "Accounts.UpdateTasks"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	type taskItem struct {
		TaskID      string `json:"taskId"`
		IsCompleted bool   `json:"isCompleted,omitempty"`
		Action      string `json:"action,omitempty"`
	}
	taskList := make([]taskItem, len(updates))
	for i, u := range updates {
		taskList[i] = taskItem{TaskID: u.TaskID, IsCompleted: u.IsCompleted, Action: u.Action}
	}
	payload := struct {
		Tasks []taskItem `json:"tasks"`
		Type  string     `json:"type,omitempty"`
	}{Tasks: taskList, Type: taskType}
	data, _ := json.Marshal(payload)
	resp, err := m.surface.generated.UpdateAccountsTasksWithBodyWithResponse(ctx, string(accountID), "application/json", bytes.NewReader(data))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// AssignTask assigns a registration task to the account.
func (m *RESTAccounts) AssignTask(ctx context.Context, accountID AccountID, task TaskAssignment) error {
	const op = "Accounts.AssignTask"
	if err := m.surface.owner.checkOpen(); err != nil {
		return err
	}
	payload := client.RegistrationTask{
		ExternalId:            task.ExternalID,
		FormName:              task.FormName,
		FormNumber:            task.FormNumber,
		IsCompleted:           task.IsCompleted,
		IsDeclined:            task.IsDeclined,
		IsRequiredForApproval: task.IsRequiredForApproval,
		QuestionIds:           task.QuestionIDs,
		State:                 task.State,
	}
	data, _ := json.Marshal(payload)
	resp, err := m.surface.generated.CreateAccountsTasksWithBodyWithResponse(ctx, string(accountID), "application/json", bytes.NewReader(data))
	if err != nil {
		e := wrapOp(op, err)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	if resp.HTTPResponse.StatusCode >= 400 {
		e := m.surface.owner.errorFrom(resp.HTTPResponse, op)
		internal.LogError(m.surface.owner.cfg.logger, e)
		return e
	}
	return nil
}

// Public types for REST accounts

type RESTAccountSummary struct {
	ID            AccountID `json:"id"`
	AccountAlias  string    `json:"accountAlias"`
	BaseCurrency  string    `json:"baseCurrency"`
	AccountTitle  string    `json:"accountTitle"`
	AccountType   string    `json:"accountType"`
	MasterAccount string    `json:"masterAccount"`
}

type LoginMessage struct {
	ID          int64     `json:"id"`
	Description string    `json:"description"`
	MessageType string    `json:"messageType"`
	State       string    `json:"state"`
	RecordDate  time.Time `json:"recordDate"`
	Username    string    `json:"username"`
	ContentID   int64     `json:"contentId"`
	Tasks       []int64   `json:"tasks"`
}

type RESTAccountStatus struct {
	AccountID       AccountID `json:"accountId"`
	AdminAccountID  string    `json:"adminAccountId"`
	DateClosed      time.Time `json:"dateClosed"`
	DateOpened      time.Time `json:"dateOpened"`
	DateStarted     time.Time `json:"dateStarted"`
	Description     string    `json:"description"`
	MasterAccountID string    `json:"masterAccountId"`
	Message         string    `json:"message"`
	State           string    `json:"state"`
	Status          string    `json:"status"`
}

type RegistrationTaskItem struct {
	TaskID                string    `json:"taskId"`
	Action                string    `json:"action"`
	DateCompleted         time.Time `json:"dateCompleted"`
	ExternalID            string    `json:"externalId"`
	FormName              string    `json:"formName"`
	FormNumber            int64     `json:"formNumber"`
	IsCompleted           bool      `json:"isCompleted"`
	IsDeclined            bool      `json:"isDeclined"`
	IsRequiredForApproval bool      `json:"isRequiredForApproval"`
	QuestionIDs           []int64   `json:"questionIds"`
	State                 string    `json:"state"`
	Warning               string    `json:"warning"`
}

type AccountConfiguration struct {
	AccountID AccountID
	Type      *string
	Value     *bool
}

type TaskUpdate struct {
	TaskID      string
	IsCompleted bool
	Action      string
}

type TaskAssignment struct {
	ExternalID            *string  `json:"externalId,omitempty"`
	FormName              *string  `json:"formName,omitempty"`
	FormNumber            *int64   `json:"formNumber,omitempty"`
	IsCompleted           *bool    `json:"isCompleted,omitempty"`
	IsDeclined            *bool    `json:"isDeclined,omitempty"`
	IsRequiredForApproval *bool    `json:"isRequiredForApproval,omitempty"`
	QuestionIDs           *[]int64 `json:"questionIds,omitempty"`
	State                 *string  `json:"state,omitempty"`
}

// Response wrapper types

type LoginMessagesWrapper struct {
	AccountID                 *string         `json:"accountId,omitempty"`
	ClearingStatus            *string         `json:"clearingStatus,omitempty"`
	ClearingStatusDescription *string         `json:"clearingStatusDescription,omitempty"`
	LoginMessagePresent       *bool           `json:"loginMessagePresent,omitempty"`
	LoginMessages             []*LoginMessage `json:"loginMessages,omitempty"`
}

type AccountStatusBulkWrapper struct {
	Accounts []*RESTAccountStatus `json:"accounts,omitempty"`
	Total    *int64               `json:"total,omitempty"`
	Offset   *int64               `json:"offset,omitempty"`
	Limit    *int64               `json:"limit,omitempty"`
}

type Au10TixWrapper struct {
	EntityID         *int64  `json:"entityId,omitempty"`
	ExternalID       *string `json:"externalId,omitempty"`
	HasError         *bool   `json:"hasError,omitempty"`
	ErrorDescription *string `json:"errorDescription,omitempty"`
	State            *string `json:"state,omitempty"`
	StartDate        *string `json:"startDate,omitempty"`
}

func (r Au10TixWrapper) KycURL() string {
	if r.ExternalID != nil {
		return *r.ExternalID
	}
	return ""
}

type RegistrationTasksWrapper struct {
	AccountID         *string                 `json:"accountId,omitempty"`
	Description       *string                 `json:"description,omitempty"`
	Empty             *bool                   `json:"empty,omitempty"`
	HasError          *bool                   `json:"hasError,omitempty"`
	ErrorDescription  *string                 `json:"errorDescription,omitempty"`
	RegistrationTasks []*RegistrationTaskItem `json:"registrationTasks,omitempty"`
	State             *string                 `json:"state,omitempty"`
	Status            *string                 `json:"status,omitempty"`
}

func (r RegistrationTasksWrapper) Tasks() []RegistrationTaskItem {
	if r.RegistrationTasks == nil {
		return nil
	}
	out := make([]RegistrationTaskItem, 0, len(r.RegistrationTasks))
	for _, t := range r.RegistrationTasks {
		if t != nil {
			out = append(out, *t)
		}
	}
	return out
}
