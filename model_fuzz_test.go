package cachestore

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
	"unicode/utf8"
)

type TestModel struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Created  int64  `json:"created"`
	Metadata string `json:"metadata,omitempty"`
}

type ComplexModel struct {
	Simple   TestModel              `json:"simple"`
	Tags     []string               `json:"tags"`
	Settings map[string]interface{} `json:"settings"`
	Nested   *NestedModel           `json:"nested,omitempty"`
}

type NestedModel struct {
	Value    float64 `json:"value"`
	Children []int   `json:"children"`
}

// FuzzSetGetModel tests SetModel/GetModel with various model types and data
func FuzzSetGetModel(f *testing.F) {
	// Seed corpus with various model scenarios
	f.Add("model1", int64(60), 123, "test name", true, int64(1609459200), "metadata1")
	f.Add("model2", int64(300), 0, "", false, int64(0), "")
	f.Add("unicode-key", int64(30), -1, "unicode name test", true, int64(-1), "unicode metadata test")
	f.Add("", int64(10), 999999, "long name that might exceed some limits", false, int64(9999999999), "")

	f.Fuzz(func(t *testing.T, key string, ttl int64, id int, name string, active bool, created int64, metadata string) {
		ctx := context.Background()

		// Skip invalid TTL values
		if ttl <= 0 || ttl > 86400 {
			t.Skip("Skipping invalid TTL")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		model := &TestModel{
			ID:       id,
			Name:     name,
			Active:   active,
			Created:  created,
			Metadata: metadata,
		}

		ttlDuration := time.Duration(ttl) * time.Second

		// SetModel operation
		err = client.SetModel(ctx, key, model, ttlDuration)
		if err != nil {
			// Some keys might be invalid
			t.Logf("SetModel failed for key %q: %v", key, err)
			return
		}

		// GetModel operation
		var retrieved TestModel
		err = client.GetModel(ctx, key, &retrieved)
		if err != nil {
			t.Errorf("GetModel failed for key %q after SetModel: %v", key, err)
			return
		}

		// Verify all fields match
		if retrieved.ID != model.ID {
			t.Errorf("ID mismatch: got %d, expected %d", retrieved.ID, model.ID)
		}

		// For string fields, handle JSON's UTF-8 sanitization
		// JSON replaces invalid UTF-8 sequences with Unicode replacement character
		if !utf8.ValidString(model.Name) {
			// If original contains invalid UTF-8, get expected result by marshaling/unmarshaling
			testModel := &TestModel{Name: model.Name}
			jsonBytes, marshalErr := json.Marshal(testModel)
			if marshalErr == nil {
				var expectedModel TestModel
				if json.Unmarshal(jsonBytes, &expectedModel) == nil {
					if retrieved.Name != expectedModel.Name {
						t.Errorf("Name with invalid UTF-8 not properly sanitized: got %q, expected %q", retrieved.Name, expectedModel.Name)
					}
				}
			}
		} else if retrieved.Name != model.Name {
			t.Errorf("Name mismatch: got %q, expected %q", retrieved.Name, model.Name)
		}

		if retrieved.Active != model.Active {
			t.Errorf("Active mismatch: got %v, expected %v", retrieved.Active, model.Active)
		}
		if retrieved.Created != model.Created {
			t.Errorf("Created mismatch: got %d, expected %d", retrieved.Created, model.Created)
		}

		// Handle metadata field UTF-8 validation
		if !utf8.ValidString(model.Metadata) {
			// If original contains invalid UTF-8, get expected result by marshaling/unmarshaling
			testModel := &TestModel{Metadata: model.Metadata}
			jsonBytes, marshalErr := json.Marshal(testModel)
			if marshalErr == nil {
				var expectedModel TestModel
				if json.Unmarshal(jsonBytes, &expectedModel) == nil {
					if retrieved.Metadata != expectedModel.Metadata {
						t.Errorf("Metadata with invalid UTF-8 not properly sanitized: got %q, expected %q", retrieved.Metadata, expectedModel.Metadata)
					}
				}
			}
		} else if retrieved.Metadata != model.Metadata {
			t.Errorf("Metadata mismatch: got %q, expected %q", retrieved.Metadata, model.Metadata)
		}

		// Test with wrong type
		var wrongType ComplexModel
		err = client.GetModel(ctx, key, &wrongType)
		if err == nil {
			// This might succeed depending on JSON structure compatibility
			t.Logf("GetModel with wrong type succeeded - this might be valid JSON unmarshaling behavior")
		}
	})
}

// FuzzComplexModelSerialization tests complex nested structures
func FuzzComplexModelSerialization(f *testing.F) {
	// Seed corpus with complex model variations
	f.Add("complex1", 1.5, "tag1", "tag2", "setting1", "value1", int64(30))
	f.Add("complex2", -999.99, "", "", "", "", int64(60))
	f.Add("unicode", 0.0, "test", "tag", "config", "value", int64(120))

	f.Fuzz(func(t *testing.T, key string, value float64, tag1, tag2, settingKey, settingValue string, ttl int64) {
		ctx := context.Background()

		if ttl <= 0 || ttl > 3600 {
			t.Skip("Skipping invalid TTL")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Build complex model
		model := &ComplexModel{
			Simple: TestModel{
				ID:       42,
				Name:     "nested simple",
				Active:   true,
				Created:  time.Now().Unix(),
				Metadata: "nested metadata",
			},
			Tags:     []string{tag1, tag2},
			Settings: make(map[string]interface{}),
			Nested: &NestedModel{
				Value:    value,
				Children: []int{1, 2, 3},
			},
		}

		// Add setting if both key and value are provided
		if settingKey != "" {
			model.Settings[settingKey] = settingValue
		}

		ttlDuration := time.Duration(ttl) * time.Second

		// SetModel operation
		err = client.SetModel(ctx, key, model, ttlDuration)
		if err != nil {
			t.Logf("SetModel failed for complex model with key %q: %v", key, err)
			return
		}

		// GetModel operation
		var retrieved ComplexModel
		err = client.GetModel(ctx, key, &retrieved)
		if err != nil {
			t.Errorf("GetModel failed for complex model with key %q: %v", key, err)
			return
		}

		// Verify complex structure
		if retrieved.Simple.ID != model.Simple.ID {
			t.Errorf("Simple.ID mismatch: got %d, expected %d", retrieved.Simple.ID, model.Simple.ID)
		}

		if len(retrieved.Tags) != len(model.Tags) {
			t.Errorf("Tags length mismatch: got %d, expected %d", len(retrieved.Tags), len(model.Tags))
		} else {
			for i, tag := range model.Tags {
				// Handle JSON's UTF-8 sanitization for tag strings
				if !utf8.ValidString(tag) {
					// If original contains invalid UTF-8, get expected result by marshaling/unmarshaling
					testTags := []string{tag}
					jsonBytes, marshalErr := json.Marshal(testTags)
					if marshalErr == nil {
						var expectedTags []string
						if json.Unmarshal(jsonBytes, &expectedTags) == nil && len(expectedTags) > 0 {
							if retrieved.Tags[i] != expectedTags[0] {
								t.Errorf("Tag[%d] with invalid UTF-8 not properly sanitized: got %q, expected %q", i, retrieved.Tags[i], expectedTags[0])
							}
						}
					}
				} else if retrieved.Tags[i] != tag {
					t.Errorf("Tag[%d] mismatch: got %q, expected %q", i, retrieved.Tags[i], tag)
				}
			}
		}

		if retrieved.Nested == nil && model.Nested != nil {
			t.Errorf("Nested model was lost during serialization")
		} else if retrieved.Nested != nil && model.Nested != nil {
			if retrieved.Nested.Value != model.Nested.Value {
				t.Errorf("Nested.Value mismatch: got %f, expected %f", retrieved.Nested.Value, model.Nested.Value)
			}
		}
	})
}

// FuzzModelJSONEdgeCases tests edge cases in JSON serialization
func FuzzModelJSONEdgeCases(f *testing.F) {
	// Seed with various JSON edge cases
	f.Add("edge1", `{"id":123,"name":"test"}`)
	f.Add("edge2", `{"invalid":"json"`)
	f.Add("edge3", `{}`)
	f.Add("edge4", `null`)
	f.Add("edge5", `{"id":"not_a_number","name":null}`)
	f.Add("unicode", `{"name":"test-key","metadata":"test-data"}`)

	f.Fuzz(func(t *testing.T, key, jsonStr string) {
		ctx := context.Background()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Try to parse as TestModel first
		var testModel TestModel
		parseErr := json.Unmarshal([]byte(jsonStr), &testModel)
		if parseErr != nil {
			t.Logf("JSON string %q is not valid TestModel JSON: %v", jsonStr, parseErr)
			return
		}

		// SetModel with parsed model
		err = client.SetModel(ctx, key, &testModel, time.Minute)
		if err != nil {
			t.Logf("SetModel failed for key %q: %v", key, err)
			return
		}

		// GetModel operation
		var retrieved TestModel
		err = client.GetModel(ctx, key, &retrieved)
		if err != nil {
			t.Errorf("GetModel failed after SetModel for key %q: %v", key, err)
			return
		}

		// Verify round-trip consistency
		originalBytes, err1 := json.Marshal(&testModel)
		retrievedBytes, err2 := json.Marshal(&retrieved)

		if err1 != nil || err2 != nil {
			t.Logf("JSON marshaling failed during verification: %v, %v", err1, err2)
			return
		}

		// Compare JSON representations (might not be exactly equal due to field ordering)
		var originalMap, retrievedMap map[string]interface{}
		if err := json.Unmarshal(originalBytes, &originalMap); err != nil {
			t.Logf("Failed to unmarshal original JSON: %v", err)
			return
		}
		if err := json.Unmarshal(retrievedBytes, &retrievedMap); err != nil {
			t.Logf("Failed to unmarshal retrieved JSON: %v", err)
			return
		}

		if !reflect.DeepEqual(originalMap, retrievedMap) {
			t.Errorf("JSON round-trip failed for key %q", key)
			t.Logf("Original: %s", originalBytes)
			t.Logf("Retrieved: %s", retrievedBytes)
		}
	})
}

// FuzzModelNilPointers tests behavior with nil pointers and empty models
func FuzzModelNilPointers(f *testing.F) {
	// Seed with various nil/empty scenarios
	f.Add("nil1", true, false)
	f.Add("nil2", false, true)
	f.Add("empty", false, false)

	f.Fuzz(func(t *testing.T, key string, useNilModel, useEmptyModel bool) {
		ctx := context.Background()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		var model interface{}

		if useNilModel {
			model = (*TestModel)(nil)
		} else if useEmptyModel {
			model = &TestModel{}
		} else {
			model = &TestModel{
				ID:      1,
				Name:    "test",
				Active:  true,
				Created: time.Now().Unix(),
			}
		}

		// SetModel should handle these cases gracefully
		err = client.SetModel(ctx, key, model, time.Minute)
		if err != nil {
			if useNilModel {
				t.Logf("SetModel with nil model failed as expected: %v", err)
				return
			}
			t.Logf("SetModel failed: %v", err)
			return
		}

		// GetModel operation
		var retrieved TestModel
		err = client.GetModel(ctx, key, &retrieved)
		if err != nil {
			t.Errorf("GetModel failed: %v", err)
			return
		}

		// For empty model, verify defaults
		if useEmptyModel {
			if retrieved.ID != 0 || retrieved.Name != "" || retrieved.Active != false {
				t.Errorf("Empty model didn't preserve zero values correctly")
			}
		}
	})
}

// FuzzModelTypeConsistency tests type consistency across different operations
func FuzzModelTypeConsistency(f *testing.F) {
	// Seed with different type scenarios
	f.Add("type1", "string", int64(123), 45.6)
	f.Add("type2", "", int64(0), 0.0)
	f.Add("unicode", "test", int64(-999), -123.456)

	f.Fuzz(func(t *testing.T, key, stringVal string, intVal int64, floatVal float64) {
		ctx := context.Background()

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Create model with mixed types
		model := map[string]interface{}{
			"string_field": stringVal,
			"int_field":    intVal,
			"float_field":  floatVal,
			"bool_field":   true,
			"nil_field":    nil,
		}

		err = client.SetModel(ctx, key, model, time.Minute)
		if err != nil {
			t.Logf("SetModel failed for mixed types: %v", err)
			return
		}

		// Retrieve as generic map
		var retrieved map[string]interface{}
		err = client.GetModel(ctx, key, &retrieved)
		if err != nil {
			t.Errorf("GetModel failed for mixed types: %v", err)
			return
		}

		// JSON unmarshaling might change types (e.g., all numbers become float64)
		// So we verify the values can be converted back

		// Handle UTF-8 validation for string fields - JSON replaces invalid UTF-8 with replacement character
		if !utf8.ValidString(stringVal) {
			// If original contains invalid UTF-8, get expected result by marshaling/unmarshaling
			testData := map[string]interface{}{"string_field": stringVal}
			jsonBytes, marshalErr := json.Marshal(testData)
			if marshalErr == nil {
				var expectedData map[string]interface{}
				if json.Unmarshal(jsonBytes, &expectedData) == nil {
					if retrieved["string_field"] != expectedData["string_field"] {
						t.Errorf("String field with invalid UTF-8 not properly sanitized: got %v, expected %v", retrieved["string_field"], expectedData["string_field"])
					}
				}
			}
		} else if retrieved["string_field"] != stringVal {
			t.Errorf("String field mismatch: got %v, expected %s", retrieved["string_field"], stringVal)
		}

		// Numbers in JSON become float64, so we need to handle conversion
		if intField, ok := retrieved["int_field"]; ok {
			switch v := intField.(type) {
			case float64:
				if int64(v) != intVal {
					t.Errorf("Int field mismatch (as float64): got %v, expected %d", v, intVal)
				}
			case int64:
				if v != intVal {
					t.Errorf("Int field mismatch: got %v, expected %d", v, intVal)
				}
			default:
				t.Logf("Unexpected type for int field: %T", v)
			}
		}
	})
}

// FuzzModelSize tests behavior with various model sizes
func FuzzModelSize(f *testing.F) {
	// Seed with different size scenarios
	f.Add("size1", 10, 5)
	f.Add("size2", 100, 50)
	f.Add("size3", 1000, 10)
	f.Add("empty", 0, 0)

	f.Fuzz(func(t *testing.T, key string, stringSize, arraySize int) {
		ctx := context.Background()

		// Limit sizes to prevent excessive memory usage
		if stringSize > 10000 || arraySize > 1000 {
			t.Skip("Skipping large sizes")
		}

		client, err := NewClient(ctx, WithFreeCache())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close(ctx)

		// Create large string
		largeString := ""
		for i := 0; i < stringSize; i++ {
			largeString += fmt.Sprintf("x%d", i%10)
		}

		// Create large array
		largeArray := make([]string, arraySize)
		for i := 0; i < arraySize; i++ {
			largeArray[i] = fmt.Sprintf("item_%d", i)
		}

		model := map[string]interface{}{
			"large_string": largeString,
			"large_array":  largeArray,
			"size_info": map[string]int{
				"string_size": stringSize,
				"array_size":  arraySize,
			},
		}

		err = client.SetModel(ctx, key, model, time.Minute)
		if err != nil {
			t.Logf("SetModel failed for large model: %v", err)
			return
		}

		var retrieved map[string]interface{}
		err = client.GetModel(ctx, key, &retrieved)
		if err != nil {
			t.Errorf("GetModel failed for large model: %v", err)
			return
		}

		// Verify large string
		if retrievedString, ok := retrieved["large_string"].(string); ok {
			if retrievedString != largeString {
				t.Errorf("Large string mismatch in length: got %d, expected %d",
					len(retrievedString), len(largeString))
			}
		}

		// Verify array size
		if retrievedArray, ok := retrieved["large_array"].([]interface{}); ok {
			if len(retrievedArray) != arraySize {
				t.Errorf("Large array size mismatch: got %d, expected %d",
					len(retrievedArray), arraySize)
			}
		}
	})
}
