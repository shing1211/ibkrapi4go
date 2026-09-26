// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package ibkr

import (
	"context"
	"testing"
)

func TestFYI_FYIDelivery(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.FYI().FYIDelivery(context.Background())
	if err != nil {
		t.Fatalf("FYIDelivery: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("delivery = %+v; want one", got)
	}
	if got[0].DeviceID != "dev-1" {
		t.Errorf("deviceId = %q; want dev-1", got[0].DeviceID)
	}
	if got[0].Type != "email" {
		t.Errorf("type = %q; want email", got[0].Type)
	}
	if got[0].Value != "user@example.com" {
		t.Errorf("value = %q; want user@example.com", got[0].Value)
	}
}

func TestFYI_ModifyFYIDelivery(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.FYI().ModifyFYIDelivery(context.Background(), "email", "user@example.com"); err != nil {
		t.Fatalf("ModifyFYIDelivery: %v", err)
	}
}

func TestFYI_ModifyFYIEmails(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.FYI().ModifyFYIEmails(context.Background(), true); err != nil {
		t.Fatalf("ModifyFYIEmails: %v", err)
	}
}

func TestFYI_DeleteFYIDevice(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.FYI().DeleteFYIDevice(context.Background(), "dev-1"); err != nil {
		t.Fatalf("DeleteFYIDevice: %v", err)
	}
}

func TestFYI_FYIDisclaimers(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.FYI().FYIDisclaimers(context.Background(), "1")
	if err != nil {
		t.Fatalf("FYIDisclaimers: %v", err)
	}
	if got.TypeCode != "1" {
		t.Errorf("typeCode = %q; want 1", got.TypeCode)
	}
	if got.Content != "Synthetic disclaimer." {
		t.Errorf("content = %q; want Synthetic disclaimer.", got.Content)
	}
}

func TestFYI_ReadFYIDisclaimer(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.FYI().ReadFYIDisclaimer(context.Background(), "1"); err != nil {
		t.Fatalf("ReadFYIDisclaimer: %v", err)
	}
}

func TestFYI_AllFYIs(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.FYI().AllFYIs(context.Background(), 10)
	if err != nil {
		t.Fatalf("AllFYIs: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("notifications = %+v; want one", got)
	}
	if got[0].ID != "1" {
		t.Errorf("id = %q; want 1", got[0].ID)
	}
	if got[0].TypeCode != "1" {
		t.Errorf("typeCode = %q; want 1", got[0].TypeCode)
	}
	if got[0].Subject != "Welcome" {
		t.Errorf("subject = %q; want Welcome", got[0].Subject)
	}
	if got[0].Read {
		t.Error("read = true; the fixture encodes false")
	}
}

func TestFYI_FYIsPager(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)
	ctx := context.Background()

	pager := cli.FYI().FYIsPager(ctx, 10)
	var seen []FYINotification
	for pager.Next(ctx) {
		seen = append(seen, pager.Value())
	}
	if err := pager.Err(); err != nil {
		t.Fatalf("pager.Err: %v", err)
	}
	if len(seen) != 1 {
		t.Fatalf("notifications = %+v; want one", seen)
	}
	if seen[0].ID != "1" {
		t.Errorf("id = %q; want 1", seen[0].ID)
	}
	if seen[0].Subject != "Welcome" {
		t.Errorf("subject = %q; want Welcome", seen[0].Subject)
	}
}

func TestFYI_ReadFYINotification(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.FYI().ReadFYINotification(context.Background(), "1"); err != nil {
		t.Fatalf("ReadFYINotification: %v", err)
	}
}

func TestFYI_FYISettings(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.FYI().FYISettings(context.Background())
	if err != nil {
		t.Fatalf("FYISettings: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("settings = %+v; want one", got)
	}
	if got[0].TypeCode != "1" {
		t.Errorf("typeCode = %q; want 1", got[0].TypeCode)
	}
	if !got[0].Enabled {
		t.Error("enabled = false; the fixture encodes true")
	}
}

func TestFYI_ModifyFYINotification(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	if err := cli.FYI().ModifyFYINotification(context.Background(), "1", true); err != nil {
		t.Fatalf("ModifyFYINotification: %v", err)
	}
}

func TestFYI_UnreadFYIs(t *testing.T) {
	gw := newGateway(t)
	cli := newTestClient(t, gw)

	got, err := cli.FYI().UnreadFYIs(context.Background())
	if err != nil {
		t.Fatalf("UnreadFYIs: %v", err)
	}
	if got.Count != 3 {
		t.Errorf("count = %d; want 3", got.Count)
	}
}
