// Copyright 2019 Google LLC
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

// Package filter defines which tickets pass which filters.  Other implementations which help
// filter tickets (eg, a range index lookup) must conform to the same set of tickets being within
// the filter as here.
package filter_test

import (
	"testing"

	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter"
	"github.com/rivalry-matchmaker/rivalry/internal/managers/filter/testcases"
	"github.com/rivalry-matchmaker/rivalry/pkg/pb"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMeetsCriteria(t *testing.T) {
	testInclusion := func(t *testing.T, pool *pb.Pool, entity filter.FilteredEntity) {
		pf, err := filter.NewPoolFilter(pool)

		require.NoError(t, err)
		require.NotNil(t, pf)

		if !pf.In(entity) {
			t.Error("entity should be included in the pool")
		}
	}

	for _, tc := range testcases.IncludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			testInclusion(t, tc.Pool, &pb.Ticket{
				SearchFields: tc.SearchFields,
				CreateTime:   ptypes.TimestampNow(),
			})
			testInclusion(t, tc.Pool, &pb.Backfill{
				SearchFields: tc.SearchFields,
				CreateTime:   ptypes.TimestampNow(),
			})
		})
	}

	testExclusion := func(t *testing.T, pool *pb.Pool, entity filter.FilteredEntity) {
		pf, err := filter.NewPoolFilter(pool)

		require.NoError(t, err)
		require.NotNil(t, pf)

		if pf.In(entity) {
			t.Error("ticket should be excluded from the pool")
		}
	}

	for _, tc := range testcases.ExcludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			testExclusion(t, tc.Pool, &pb.Ticket{
				SearchFields: tc.SearchFields,
				CreateTime:   ptypes.TimestampNow(),
			})
			testExclusion(t, tc.Pool, &pb.Backfill{
				SearchFields: tc.SearchFields,
				CreateTime:   ptypes.TimestampNow(),
			})
		})
	}
}

func TestValidPoolFilter(t *testing.T) {
	for _, tc := range []struct {
		name string
		pool *pb.Pool
		code codes.Code
		msg  string
	}{
		{
			"invalid create before",
			&pb.Pool{
				CreatedBefore: &timestamp.Timestamp{Nanos: -1},
			},
			codes.InvalidArgument,
			".invalid created_before value",
		},
		{
			"invalid create after",
			&pb.Pool{
				CreatedAfter: &timestamp.Timestamp{Nanos: -1},
			},
			codes.InvalidArgument,
			".invalid created_after value",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pf, err := filter.NewPoolFilter(tc.pool)

			require.Error(t, err)
			require.Nil(t, pf)

			s := status.Convert(err)
			require.Equal(t, tc.code, s.Code())
			require.Equal(t, tc.msg, s.Message())
		})
	}
}
