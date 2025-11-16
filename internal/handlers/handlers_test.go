package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"laguna/internal/mocks"
	"testing"

	"laguna/internal/query"

	"github.com/golang/mock/gomock"
)

var ctx = context.Background()

func TestUniversalHandler_Handle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := mocks.MockLogger{}
	mockQB := mocks.NewMockQueryBuilder(ctrl)
	mockDB := mocks.NewMockDatabase(ctrl)

	handler := NewUniversalHandler(mockQB, mockDB, &log)

	tests := []struct {
		name       string
		input      string
		qbErr      error
		dbResp     string
		dbErr      error
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "successful get request",
			input:      "GET key\n",
			dbResp:     "value",
			wantOutput: "value\n",
		},
		{
			name:       "successful set request",
			input:      "SET key value\n",
			dbResp:     "OK",
			wantOutput: "OK\n",
		},
		{
			name:       "successful del request",
			input:      "DEL key\n",
			dbResp:     "OK",
			wantOutput: "OK\n",
		},
		{
			name:       "successful get miss request",
			input:      "GET key\n",
			dbResp:     "",
			wantOutput: "\n",
		},
		{
			name:       "parse error",
			input:      "BAD QUERY\n",
			qbErr:      errors.New("parse failed"),
			wantOutput: "parse failed\n",
		},
		{
			name:       "db execute error",
			input:      "GET key\n",
			dbErr:      errors.New("execute failed"),
			wantOutput: "execute failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bufIn := bytes.NewBufferString(tt.input)
			bufOut := &bytes.Buffer{}

			q := query.Query{}

			if tt.qbErr != nil {
				mockQB.EXPECT().Parse([]byte(tt.input)).Return(query.Query{}, tt.qbErr)
			} else {
				mockQB.EXPECT().Parse([]byte(tt.input)).Return(q, nil)
			}

			if tt.dbErr != nil {
				mockDB.EXPECT().Execute(ctx, q).Return("", tt.dbErr)
			} else if tt.dbResp != "" && tt.qbErr == nil {
				mockDB.EXPECT().Execute(ctx, q).Return(tt.dbResp, nil)
			} else if tt.dbResp == "" && tt.qbErr == nil {
				mockDB.EXPECT().Execute(ctx, q).Return("", nil)
			}

			err := handler.Handle(ctx, bufIn, bufOut)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if bufOut.String() != tt.wantOutput {
				t.Errorf("unexpected output: got %q, want %q", bufOut.String(), tt.wantOutput)
			}
		})
	}
}

func TestUniversalHandler_Handle_EOF(t *testing.T) {
	log := mocks.MockLogger{}

	handler := NewUniversalHandler(nil, nil, &log)

	bufIn := bytes.NewBufferString("")
	bufOut := &bytes.Buffer{}

	err := handler.Handle(ctx, bufIn, bufOut)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got: %v", err)
	}

	if bufOut.Len() != 0 {
		t.Errorf("expected empty output, got: %q", bufOut.String())
	}
}
