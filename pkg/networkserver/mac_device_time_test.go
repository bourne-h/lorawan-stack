// Copyright © 2018 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package networkserver

import (
	"testing"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/smartystreets/assertions"
	"go.thethings.network/lorawan-stack/pkg/ttnpb"
	"go.thethings.network/lorawan-stack/pkg/util/test"
	"go.thethings.network/lorawan-stack/pkg/util/test/assertions/should"
)

func TestHandleDeviceTimeReq(t *testing.T) {
	for _, tc := range []struct {
		Name             string
		Device, Expected *ttnpb.EndDevice
		Message          *ttnpb.UplinkMessage
		Error            error
	}{
		{
			Name:   "empty queue",
			Device: &ttnpb.EndDevice{},
			Expected: &ttnpb.EndDevice{
				QueuedMACCommands: []*ttnpb.MACCommand{
					(&ttnpb.MACCommand_DeviceTimeAns{
						Time: time.Unix(42, 42),
					}).MACCommand(),
				},
			},
			Message: &ttnpb.UplinkMessage{
				ReceivedAt: time.Unix(42, 42),
			},
			Error: nil,
		},
		{
			Name: "non-empty queue",
			Device: &ttnpb.EndDevice{
				QueuedMACCommands: []*ttnpb.MACCommand{
					(&ttnpb.MACCommand_DeviceTimeAns{
						Time: time.Unix(42, 0),
					}).MACCommand(),
				},
			},
			Expected: &ttnpb.EndDevice{
				QueuedMACCommands: []*ttnpb.MACCommand{
					(&ttnpb.MACCommand_DeviceTimeAns{
						Time: time.Unix(42, 0),
					}).MACCommand(),
					(&ttnpb.MACCommand_DeviceTimeAns{
						Time: time.Unix(42, 42),
					}).MACCommand(),
				},
			},
			Message: &ttnpb.UplinkMessage{
				ReceivedAt: time.Unix(42, 42),
			},
			Error: nil,
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			a := assertions.New(t)

			dev := deepcopy.Copy(tc.Device).(*ttnpb.EndDevice)

			err := handleDeviceTimeReq(test.Context(), dev, tc.Message)
			if tc.Error != nil {
				a.So(err, should.BeError)
				return
			}

			a.So(err, should.BeNil)
			a.So(dev, should.Resemble, tc.Expected)
		})
	}
}