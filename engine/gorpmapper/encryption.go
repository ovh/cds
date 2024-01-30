package gorpmapper

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	KeyEcnryptionIdentifier = "db-crypt"
)

func (m *Mapper) Encrypt(src interface{}, dst *[]byte, extra []interface{}) error {
	clearContent, err := json.Marshal(src)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to marshal content: %v", err))
	}

	var extrabytes [][]byte
	for _, e := range extra {
		btes, _ := json.Marshal(e)
		extrabytes = append(extrabytes, btes)
	}

	btes, err := m.encryptionKey.Encrypt(clearContent, extrabytes...)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to encrypt content: %v", err))
	}

	*dst = btes

	return nil
}

func (m *Mapper) Decrypt(src []byte, dest interface{}, extra []interface{}) error {
	t := reflect.TypeOf(dest)
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("gorpmapping: cannot Decrypt into a non-pointer : %v", t)
	}

	var extrabytes [][]byte
	for _, e := range extra {
		btes, _ := json.Marshal(e)
		extrabytes = append(extrabytes, btes)
	}

	clearContent, err := m.encryptionKey.Decrypt(src, extrabytes...)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to decrypt content: %v", err))
	}

	return sdk.JSONUnmarshal(clearContent, dest)
}

func (m *Mapper) updateEncryptedData(db gorp.SqlExecutor, i interface{}) error {
	mapping, has := m.GetTableMapping(i)
	if !has {
		return sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}

	if !mapping.EncryptedEntity {
		return nil
	}

	val := reflect.ValueOf(i)
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var encryptedContents = make(map[string][]byte, len(mapping.EncryptedFields))
	var encryptedColumns = make(map[string]string, len(mapping.EncryptedFields))

	for _, f := range mapping.EncryptedFields {
		var encryptedContent []byte
		field := val.FieldByName(f.Name).Interface()

		var extras []interface{}
		for _, extra := range f.Extras {
			fieldV := val.FieldByName(extra)
			if !fieldV.IsValid() {
				return sdk.WithStack(fmt.Errorf("unable to find extra field %s", extra))
			}
			extras = append(extras, fieldV.Interface())
		}
		if err := m.Encrypt(&field, &encryptedContent, extras); err != nil {
			return err
		}
		encryptedContents[f.Name] = encryptedContent
		encryptedColumns[f.Name] = f.Column
	}

	table, key, id, err := m.dbMappingPKey(i)
	if err != nil {
		return sdk.WrapError(err, "primary key field not found in table: %s", table)
	}

	var updateSlice []string
	var encryptedContentArgs []interface{}
	var c = 1
	for f := range encryptedContents {
		encryptedContentArgs = append(encryptedContentArgs, encryptedContents[f])
		updateSlice = append(updateSlice, encryptedColumns[f]+" = $"+strconv.Itoa(c))
		c++
	}
	encryptedContentArgs = append(encryptedContentArgs, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = $%d", table, strings.Join(updateSlice, ","), key, c)
	res, err := db.Exec(query, encryptedContentArgs...)
	if err != nil {
		return sdk.WithStack(err)
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return sdk.WithStack(fmt.Errorf("invalid encrypted query: %s", query))
	}
	return nil
}

func (m *Mapper) resetEncryptedData(db gorp.SqlExecutor, i interface{}) error {
	mapping, has := m.GetTableMapping(i)
	if !has {
		return sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}
	if !mapping.EncryptedEntity {
		return nil
	}

	val := reflect.ValueOf(i)
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for _, f := range mapping.EncryptedFields {
		// Reset the field to the zero value of the placeholder
		field := val.FieldByName(f.Name)
		if field.Kind() == reflect.String {
			placeholder := reflect.ValueOf(sdk.PasswordPlaceholder)
			field.Set(placeholder)
		} else {
			placeholder := reflect.Zero(field.Type())
			field.Set(placeholder)
		}
	}
	return nil
}

func getEncryptedData(ctx context.Context, m *Mapper, db gorp.SqlExecutor, i interface{}) error {
	_, end := telemetry.Span(ctx, "gorpmappeer.getEncryptedData")
	defer end()

	// If the target is a slice or a pointer of slice, let's call getEncryptedSliceData
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		if t = t.Elem(); t.Kind() == reflect.Slice {
			return m.getEncryptedSliceData(ctx, db, i)
		}
	} else if t.Kind() == reflect.Slice {
		return m.getEncryptedSliceData(ctx, db, i)
	}

	// Get the TableMapping for the concrete type. If the type entity is not encrypt, let's skip all the things
	mapping, has := m.GetTableMapping(i)
	if !has {
		return sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}
	if !mapping.EncryptedEntity {
		return nil
	}

	table, key, id, err := m.dbMappingPKey(i)
	if err != nil {
		return sdk.WrapError(err, "primary key field not found in table: %s", table)
	}

	var encryptedColumnsSlice = make([]string, len(mapping.EncryptedFields))
	var fieldsValue = make(map[int]*reflect.Value, len(mapping.EncryptedFields))
	var encryptedContents = make([]interface{}, len(mapping.EncryptedFields))
	var extrasFieldsNames = make(map[int][]string, len(mapping.EncryptedFields))

	val := reflect.ValueOf(i)
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for idx, f := range mapping.EncryptedFields {
		dbTag := reflectFindFieldTagValue(i, f.Name, "db")
		column := strings.SplitN(dbTag, ",", 2)[0]

		encryptedColumnsSlice[idx] = column
		var encryptedContent []byte
		encryptedContents[idx] = &encryptedContent

		field := val.FieldByName(f.Name)
		fieldsValue[idx] = &field
		extrasFieldsNames[idx] = f.Extras
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", strings.Join(encryptedColumnsSlice, ","), table, key)
	if err := db.QueryRow(query, id).Scan(encryptedContents...); err != nil {
		return sdk.WrapError(err, "query: %s", query)
	}

	// Loop over the loaded encrypted content
	// Decrypt them and set the target fields with the result
	for idx, encryptedContent := range encryptedContents {
		extrasFieldNames := extrasFieldsNames[idx]
		var extras []interface{}
		for _, e := range extrasFieldNames {
			extras = append(extras, val.FieldByName(e).Interface())
		}

		var encryptedContent = encryptedContent.(*[]byte)
		var targetField = val.FieldByName(mapping.EncryptedFields[idx].Name)
		var targetHolder = reflect.New(reflect.TypeOf(targetField.Interface())).Interface()

		if err := m.Decrypt(*encryptedContent, targetHolder, extras); err != nil {
			return err
		}
		fieldsValue[idx].Set(reflect.ValueOf(targetHolder).Elem())
	}

	return nil
}

func (m *Mapper) getEncryptedSliceData(ctx context.Context, db gorp.SqlExecutor, i interface{}) error {
	_, end := telemetry.Span(ctx, "gorpmappeer.getEncryptedSliceData")
	defer end()

	// Let's consider the value pointed only
	val := reflect.ValueOf(i)
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Lookup the contained type of the slice
	eval := reflect.TypeOf(val.Interface()).Elem()
	ieval := reflect.New(eval).Interface()

	// Find the tabble mapping for the contained type of the slice
	mapping, has := m.GetTableMapping(ieval)
	if !has {
		return sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}
	// If the entity is not encrypted, let's skip all the things
	if !mapping.EncryptedEntity {
		return nil
	}

	table, key, pk, err := m.dbMappingPKey(ieval)
	if err != nil {
		return sdk.WrapError(err, "primary key field not found in table: %s", table)
	}

	// We need to gather all primary keys value for all elements of the targets slice
	var pks = make([]interface{}, val.Len())
	for idx := range pks {
		pks[idx] = reflectFindValueByTag(val.Index(idx).Interface(), "db", key)
	}

	// We prepare the list of all the fields we want to collect from the database
	var encryptedColumnsSlice = make([]string, len(mapping.EncryptedFields))
	for idx, f := range mapping.EncryptedFields {
		dbTag := reflectFindFieldTagValue(ieval, f.Name, "db")
		column := strings.SplitN(dbTag, ",", 2)[0]
		encryptedColumnsSlice[idx] = column
	}

	// We load the primary key and the encrypted content from the database
	query := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s::text = ANY(string_to_array($1, ',')::text[])", key, strings.Join(encryptedColumnsSlice, ","), table, key)
	rows, err := db.Query(query, ToQueryString(pks))
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close() // nolint

	// For each row returned by the query, we will scan the result to get the primary key and all the encrypted columns
	// We need the primary key to find out which element of the targeted slice (pass in parameter of the fonction) we have to update
	var scanner = make([]interface{}, len(mapping.EncryptedFields)+1)
	for rows.Next() {
		// Step 1. Prepare the scanner
		// Prepare a holder for the primary key
		newPk := reflect.New(reflect.TypeOf(pk)).Interface()
		scanner[0] = &newPk
		// Prepare holders for encrypted contents
		for idx := range mapping.EncryptedFields {
			var encryptedContent []byte
			scanner[idx+1] = &encryptedContent
		}
		if err := rows.Scan(scanner...); err != nil {
			return sdk.WithStack(err)
		}

		// Step 2. Locate the element of the targeted slice we want to update
		var targetSliceFound bool
		for idxTargetSlice := 0; idxTargetSlice < val.Len(); idxTargetSlice++ {
			targetSlice := val.Index(idxTargetSlice)

			// Find the right target against the primary key
			primaryKeyReference := reflectFindValueByTag(targetSlice.Interface(), "db", key)

			// Check the primary key known the from target slice and from the database
			// the pk could be a []uint8. If it's the case, string values are compared
			if reflect.DeepEqual(primaryKeyReference, newPk) || primaryKeyReference == fmt.Sprintf("%s", newPk) {
				targetSliceFound = true
				// Decrypt all the contents
				for idx := range mapping.EncryptedFields {
					// We gather the extras fields for each encrypted field
					var extras []interface{}
					for _, e := range mapping.EncryptedFields[idx].Extras {
						extras = append(extras, targetSlice.FieldByName(e).Interface())
					}
					targetField := targetSlice.FieldByName(mapping.EncryptedFields[idx].Name)
					encryptedContentPtr := scanner[idx+1]
					encryptedContent := reflect.ValueOf(encryptedContentPtr).Elem().Interface().([]byte)
					targetHolder := reflect.New(reflect.TypeOf(targetField.Interface())).Interface()
					// Decrypt and store the result in the target slice element through power of reflection
					if err := m.Decrypt(encryptedContent, targetHolder, extras); err != nil {
						return sdk.WithStack(err)
					}
					targetField.Set(reflect.ValueOf(targetHolder).Elem())
				}
			}
		}
		if !targetSliceFound {
			return sdk.WithStack(fmt.Errorf("unmatched element with pk type:%T - value:%v - pks:%v", newPk, newPk, pks))
		}
	}

	return nil
}
