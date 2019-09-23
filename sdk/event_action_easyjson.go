// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package sdk

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjson5e9de23dDecodeGithubComOvhCdsSdk(in *jlexer.Lexer, out *EventActionUpdate) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "OldAction":
			easyjson5e9de23dDecodeGithubComOvhCdsSdk1(in, &out.OldAction)
		case "NewAction":
			easyjson5e9de23dDecodeGithubComOvhCdsSdk1(in, &out.NewAction)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk(out *jwriter.Writer, in EventActionUpdate) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"OldAction\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson5e9de23dEncodeGithubComOvhCdsSdk1(out, in.OldAction)
	}
	{
		const prefix string = ",\"NewAction\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson5e9de23dEncodeGithubComOvhCdsSdk1(out, in.NewAction)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v EventActionUpdate) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson5e9de23dEncodeGithubComOvhCdsSdk(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v EventActionUpdate) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson5e9de23dEncodeGithubComOvhCdsSdk(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *EventActionUpdate) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson5e9de23dDecodeGithubComOvhCdsSdk(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *EventActionUpdate) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson5e9de23dDecodeGithubComOvhCdsSdk(l, v)
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk1(in *jlexer.Lexer, out *Action) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = int64(in.Int64())
		case "group_id":
			if in.IsNull() {
				in.Skip()
				out.GroupID = nil
			} else {
				if out.GroupID == nil {
					out.GroupID = new(int64)
				}
				*out.GroupID = int64(in.Int64())
			}
		case "name":
			out.Name = string(in.String())
		case "type":
			out.Type = string(in.String())
		case "description":
			out.Description = string(in.String())
		case "enabled":
			out.Enabled = bool(in.Bool())
		case "deprecated":
			out.Deprecated = bool(in.Bool())
		case "step_name":
			out.StepName = string(in.String())
		case "optional":
			out.Optional = bool(in.Bool())
		case "always_executed":
			out.AlwaysExecuted = bool(in.Bool())
		case "requirements":
			if in.IsNull() {
				in.Skip()
				out.Requirements = nil
			} else {
				in.Delim('[')
				if out.Requirements == nil {
					if !in.IsDelim(']') {
						out.Requirements = make(RequirementList, 0, 1)
					} else {
						out.Requirements = RequirementList{}
					}
				} else {
					out.Requirements = (out.Requirements)[:0]
				}
				for !in.IsDelim(']') {
					var v1 Requirement
					if data := in.Raw(); in.Ok() {
						in.AddError((v1).UnmarshalJSON(data))
					}
					out.Requirements = append(out.Requirements, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "parameters":
			if in.IsNull() {
				in.Skip()
				out.Parameters = nil
			} else {
				in.Delim('[')
				if out.Parameters == nil {
					if !in.IsDelim(']') {
						out.Parameters = make([]Parameter, 0, 1)
					} else {
						out.Parameters = []Parameter{}
					}
				} else {
					out.Parameters = (out.Parameters)[:0]
				}
				for !in.IsDelim(']') {
					var v2 Parameter
					easyjson5e9de23dDecodeGithubComOvhCdsSdk2(in, &v2)
					out.Parameters = append(out.Parameters, v2)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "actions":
			if in.IsNull() {
				in.Skip()
				out.Actions = nil
			} else {
				in.Delim('[')
				if out.Actions == nil {
					if !in.IsDelim(']') {
						out.Actions = make([]Action, 0, 1)
					} else {
						out.Actions = []Action{}
					}
				} else {
					out.Actions = (out.Actions)[:0]
				}
				for !in.IsDelim(']') {
					var v3 Action
					easyjson5e9de23dDecodeGithubComOvhCdsSdk1(in, &v3)
					out.Actions = append(out.Actions, v3)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "group":
			if in.IsNull() {
				in.Skip()
				out.Group = nil
			} else {
				if out.Group == nil {
					out.Group = new(Group)
				}
				easyjson5e9de23dDecodeGithubComOvhCdsSdk3(in, out.Group)
			}
		case "first_audit":
			if in.IsNull() {
				in.Skip()
				out.FirstAudit = nil
			} else {
				if out.FirstAudit == nil {
					out.FirstAudit = new(AuditAction)
				}
				easyjson5e9de23dDecodeGithubComOvhCdsSdk4(in, out.FirstAudit)
			}
		case "last_audit":
			if in.IsNull() {
				in.Skip()
				out.LastAudit = nil
			} else {
				if out.LastAudit == nil {
					out.LastAudit = new(AuditAction)
				}
				easyjson5e9de23dDecodeGithubComOvhCdsSdk4(in, out.LastAudit)
			}
		case "editable":
			out.Editable = bool(in.Bool())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk1(out *jwriter.Writer, in Action) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ID))
	}
	if in.GroupID != nil {
		const prefix string = ",\"group_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(*in.GroupID))
	}
	{
		const prefix string = ",\"name\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Name))
	}
	{
		const prefix string = ",\"type\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Type))
	}
	{
		const prefix string = ",\"description\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Description))
	}
	{
		const prefix string = ",\"enabled\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Enabled))
	}
	{
		const prefix string = ",\"deprecated\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Deprecated))
	}
	if in.StepName != "" {
		const prefix string = ",\"step_name\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.StepName))
	}
	{
		const prefix string = ",\"optional\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Optional))
	}
	{
		const prefix string = ",\"always_executed\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.AlwaysExecuted))
	}
	{
		const prefix string = ",\"requirements\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Requirements == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v4, v5 := range in.Requirements {
				if v4 > 0 {
					out.RawByte(',')
				}
				out.Raw((v5).MarshalJSON())
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"parameters\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Parameters == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v6, v7 := range in.Parameters {
				if v6 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk2(out, v7)
			}
			out.RawByte(']')
		}
	}
	if len(in.Actions) != 0 {
		const prefix string = ",\"actions\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v8, v9 := range in.Actions {
				if v8 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk1(out, v9)
			}
			out.RawByte(']')
		}
	}
	if in.Group != nil {
		const prefix string = ",\"group\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson5e9de23dEncodeGithubComOvhCdsSdk3(out, *in.Group)
	}
	if in.FirstAudit != nil {
		const prefix string = ",\"first_audit\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson5e9de23dEncodeGithubComOvhCdsSdk4(out, *in.FirstAudit)
	}
	if in.LastAudit != nil {
		const prefix string = ",\"last_audit\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson5e9de23dEncodeGithubComOvhCdsSdk4(out, *in.LastAudit)
	}
	if in.Editable {
		const prefix string = ",\"editable\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Editable))
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk4(in *jlexer.Lexer, out *AuditAction) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "action_id":
			out.ActionID = int64(in.Int64())
		case "data_type":
			out.DataType = string(in.String())
		case "data_before":
			out.DataBefore = string(in.String())
		case "data_after":
			out.DataAfter = string(in.String())
		case "id":
			out.ID = int64(in.Int64())
		case "triggered_by":
			out.TriggeredBy = string(in.String())
		case "created":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.Created).UnmarshalJSON(data))
			}
		case "event_type":
			out.EventType = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk4(out *jwriter.Writer, in AuditAction) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"action_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ActionID))
	}
	{
		const prefix string = ",\"data_type\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DataType))
	}
	{
		const prefix string = ",\"data_before\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DataBefore))
	}
	{
		const prefix string = ",\"data_after\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.DataAfter))
	}
	{
		const prefix string = ",\"id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ID))
	}
	{
		const prefix string = ",\"triggered_by\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.TriggeredBy))
	}
	{
		const prefix string = ",\"created\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Raw((in.Created).MarshalJSON())
	}
	{
		const prefix string = ",\"event_type\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.EventType))
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk3(in *jlexer.Lexer, out *Group) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = int64(in.Int64())
		case "name":
			out.Name = string(in.String())
		case "admins":
			if in.IsNull() {
				in.Skip()
				out.Admins = nil
			} else {
				in.Delim('[')
				if out.Admins == nil {
					if !in.IsDelim(']') {
						out.Admins = make([]User, 0, 1)
					} else {
						out.Admins = []User{}
					}
				} else {
					out.Admins = (out.Admins)[:0]
				}
				for !in.IsDelim(']') {
					var v10 User
					easyjson5e9de23dDecodeGithubComOvhCdsSdk5(in, &v10)
					out.Admins = append(out.Admins, v10)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "users":
			if in.IsNull() {
				in.Skip()
				out.Users = nil
			} else {
				in.Delim('[')
				if out.Users == nil {
					if !in.IsDelim(']') {
						out.Users = make([]User, 0, 1)
					} else {
						out.Users = []User{}
					}
				} else {
					out.Users = (out.Users)[:0]
				}
				for !in.IsDelim(']') {
					var v11 User
					easyjson5e9de23dDecodeGithubComOvhCdsSdk5(in, &v11)
					out.Users = append(out.Users, v11)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "tokens":
			if in.IsNull() {
				in.Skip()
				out.Tokens = nil
			} else {
				in.Delim('[')
				if out.Tokens == nil {
					if !in.IsDelim(']') {
						out.Tokens = make([]Token, 0, 1)
					} else {
						out.Tokens = []Token{}
					}
				} else {
					out.Tokens = (out.Tokens)[:0]
				}
				for !in.IsDelim(']') {
					var v12 Token
					easyjson5e9de23dDecodeGithubComOvhCdsSdk6(in, &v12)
					out.Tokens = append(out.Tokens, v12)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk3(out *jwriter.Writer, in Group) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ID))
	}
	{
		const prefix string = ",\"name\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Name))
	}
	if len(in.Admins) != 0 {
		const prefix string = ",\"admins\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v13, v14 := range in.Admins {
				if v13 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk5(out, v14)
			}
			out.RawByte(']')
		}
	}
	if len(in.Users) != 0 {
		const prefix string = ",\"users\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v15, v16 := range in.Users {
				if v15 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk5(out, v16)
			}
			out.RawByte(']')
		}
	}
	if len(in.Tokens) != 0 {
		const prefix string = ",\"tokens\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v17, v18 := range in.Tokens {
				if v17 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk6(out, v18)
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk6(in *jlexer.Lexer, out *Token) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = int64(in.Int64())
		case "group_id":
			out.GroupID = int64(in.Int64())
		case "group_name":
			out.GroupName = string(in.String())
		case "token":
			out.Token = string(in.String())
		case "description":
			out.Description = string(in.String())
		case "creator":
			out.Creator = string(in.String())
		case "expiration":
			out.Expiration = Expiration(in.Int())
		case "created":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.Created).UnmarshalJSON(data))
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk6(out *jwriter.Writer, in Token) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ID))
	}
	{
		const prefix string = ",\"group_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.GroupID))
	}
	{
		const prefix string = ",\"group_name\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.GroupName))
	}
	{
		const prefix string = ",\"token\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Token))
	}
	{
		const prefix string = ",\"description\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Description))
	}
	{
		const prefix string = ",\"creator\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Creator))
	}
	{
		const prefix string = ",\"expiration\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Expiration))
	}
	{
		const prefix string = ",\"created\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Raw((in.Created).MarshalJSON())
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk5(in *jlexer.Lexer, out *User) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = int64(in.Int64())
		case "username":
			out.Username = string(in.String())
		case "fullname":
			out.Fullname = string(in.String())
		case "email":
			out.Email = string(in.String())
		case "admin":
			out.Admin = bool(in.Bool())
		case "groups":
			if in.IsNull() {
				in.Skip()
				out.Groups = nil
			} else {
				in.Delim('[')
				if out.Groups == nil {
					if !in.IsDelim(']') {
						out.Groups = make([]Group, 0, 1)
					} else {
						out.Groups = []Group{}
					}
				} else {
					out.Groups = (out.Groups)[:0]
				}
				for !in.IsDelim(']') {
					var v19 Group
					easyjson5e9de23dDecodeGithubComOvhCdsSdk3(in, &v19)
					out.Groups = append(out.Groups, v19)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "origin":
			out.Origin = string(in.String())
		case "favorites":
			if in.IsNull() {
				in.Skip()
				out.Favorites = nil
			} else {
				in.Delim('[')
				if out.Favorites == nil {
					if !in.IsDelim(']') {
						out.Favorites = make([]Favorite, 0, 1)
					} else {
						out.Favorites = []Favorite{}
					}
				} else {
					out.Favorites = (out.Favorites)[:0]
				}
				for !in.IsDelim(']') {
					var v20 Favorite
					easyjson5e9de23dDecodeGithubComOvhCdsSdk7(in, &v20)
					out.Favorites = append(out.Favorites, v20)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "permissions":
			if data := in.Raw(); in.Ok() {
				in.AddError((out.Permissions).UnmarshalJSON(data))
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk5(out *jwriter.Writer, in User) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ID))
	}
	{
		const prefix string = ",\"username\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Username))
	}
	{
		const prefix string = ",\"fullname\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Fullname))
	}
	{
		const prefix string = ",\"email\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Email))
	}
	{
		const prefix string = ",\"admin\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Admin))
	}
	if len(in.Groups) != 0 {
		const prefix string = ",\"groups\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		{
			out.RawByte('[')
			for v21, v22 := range in.Groups {
				if v21 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk3(out, v22)
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"origin\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Origin))
	}
	{
		const prefix string = ",\"favorites\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Favorites == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v23, v24 := range in.Favorites {
				if v23 > 0 {
					out.RawByte(',')
				}
				easyjson5e9de23dEncodeGithubComOvhCdsSdk7(out, v24)
			}
			out.RawByte(']')
		}
	}
	if true {
		const prefix string = ",\"permissions\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Raw((in.Permissions).MarshalJSON())
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk7(in *jlexer.Lexer, out *Favorite) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "project_ids":
			if in.IsNull() {
				in.Skip()
				out.ProjectIDs = nil
			} else {
				in.Delim('[')
				if out.ProjectIDs == nil {
					if !in.IsDelim(']') {
						out.ProjectIDs = make([]int64, 0, 8)
					} else {
						out.ProjectIDs = []int64{}
					}
				} else {
					out.ProjectIDs = (out.ProjectIDs)[:0]
				}
				for !in.IsDelim(']') {
					var v25 int64
					v25 = int64(in.Int64())
					out.ProjectIDs = append(out.ProjectIDs, v25)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "workflow_ids":
			if in.IsNull() {
				in.Skip()
				out.WorkflowIDs = nil
			} else {
				in.Delim('[')
				if out.WorkflowIDs == nil {
					if !in.IsDelim(']') {
						out.WorkflowIDs = make([]int64, 0, 8)
					} else {
						out.WorkflowIDs = []int64{}
					}
				} else {
					out.WorkflowIDs = (out.WorkflowIDs)[:0]
				}
				for !in.IsDelim(']') {
					var v26 int64
					v26 = int64(in.Int64())
					out.WorkflowIDs = append(out.WorkflowIDs, v26)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk7(out *jwriter.Writer, in Favorite) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"project_ids\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.ProjectIDs == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v27, v28 := range in.ProjectIDs {
				if v27 > 0 {
					out.RawByte(',')
				}
				out.Int64(int64(v28))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"workflow_ids\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.WorkflowIDs == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v29, v30 := range in.WorkflowIDs {
				if v29 > 0 {
					out.RawByte(',')
				}
				out.Int64(int64(v30))
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk2(in *jlexer.Lexer, out *Parameter) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "id":
			out.ID = int64(in.Int64())
		case "name":
			out.Name = string(in.String())
		case "type":
			out.Type = string(in.String())
		case "value":
			out.Value = string(in.String())
		case "description":
			out.Description = string(in.String())
		case "advanced":
			out.Advanced = bool(in.Bool())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk2(out *jwriter.Writer, in Parameter) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.ID))
	}
	{
		const prefix string = ",\"name\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Name))
	}
	{
		const prefix string = ",\"type\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Type))
	}
	{
		const prefix string = ",\"value\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Value))
	}
	if in.Description != "" {
		const prefix string = ",\"description\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Description))
	}
	if in.Advanced {
		const prefix string = ",\"advanced\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Advanced))
	}
	out.RawByte('}')
}
func easyjson5e9de23dDecodeGithubComOvhCdsSdk8(in *jlexer.Lexer, out *EventActionAdd) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "Action":
			easyjson5e9de23dDecodeGithubComOvhCdsSdk1(in, &out.Action)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson5e9de23dEncodeGithubComOvhCdsSdk8(out *jwriter.Writer, in EventActionAdd) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"Action\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		easyjson5e9de23dEncodeGithubComOvhCdsSdk1(out, in.Action)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v EventActionAdd) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson5e9de23dEncodeGithubComOvhCdsSdk8(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v EventActionAdd) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson5e9de23dEncodeGithubComOvhCdsSdk8(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *EventActionAdd) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson5e9de23dDecodeGithubComOvhCdsSdk8(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *EventActionAdd) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson5e9de23dDecodeGithubComOvhCdsSdk8(l, v)
}
