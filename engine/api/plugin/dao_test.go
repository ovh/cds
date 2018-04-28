package plugin

import (
	"reflect"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

func TestInsert(t *testing.T) {
	type args struct {
		db gorp.SqlExecutor
		p  *sdk.GRPCPlugin
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Insert(tt.args.db, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		db gorp.SqlExecutor
		p  *sdk.GRPCPlugin
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Update(tt.args.db, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type args struct {
		db gorp.SqlExecutor
		p  *sdk.GRPCPlugin
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Delete(tt.args.db, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadByName(t *testing.T) {
	type args struct {
		db   gorp.SqlExecutor
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    *sdk.GRPCPlugin
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadByName(tt.args.db, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadAll(t *testing.T) {
	type args struct {
		db gorp.SqlExecutor
	}
	tests := []struct {
		name    string
		args    args
		want    []sdk.GRPCPlugin
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadAll(tt.args.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadAll() = %v, want %v", got, tt.want)
			}
		})
	}
}
