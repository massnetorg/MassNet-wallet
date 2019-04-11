// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
)

type NotificationType int

type NotificationCallback func(*Notification)

const (
	NTBlockAccepted NotificationType = iota

	NTBlockConnected

	NTBlockDisconnected
)

// notificationTypeStrings is a map of notification types back to their constant
// names for pretty printing.
var notificationTypeStrings = map[NotificationType]string{
	NTBlockAccepted:     "NTBlockAccepted",
	NTBlockConnected:    "NTBlockConnected",
	NTBlockDisconnected: "NTBlockDisconnected",
}

// String returns the NotificationType in human-readable form.
func (n NotificationType) String() string {
	if s, ok := notificationTypeStrings[n]; ok {
		return s
	}
	return fmt.Sprintf("Unknown Notification Type (%d)", int(n))
}

type Notification struct {
	Type NotificationType
	Data interface{}
}

// sendNotification sends a notification with the passed type and data if the
// caller requested notifications by providing a callback function in the call
// to New.
func (b *BlockChain) sendNotification(typ NotificationType, data interface{}) {
	// Ignore it if the caller didn't request notifications.
	if b.notifications == nil {
		return
	}

	// Generate and send the notification.
	n := Notification{Type: typ, Data: data}
	b.notifications(&n)
}
