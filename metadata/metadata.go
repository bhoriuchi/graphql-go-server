package metadata

import "context"

var MetadataKey interface{} = "metadata"

// New creates a new metadata context
func New() context.Context {
	return context.WithValue(context.Background(), MetadataKey, map[string]interface{}{})
}

// NewWithContext creates a new metadata context from an existing one
func NewWithContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, MetadataKey, map[string]interface{}{})
}

// getMetadata fetches the metadata map from the context
func getMetadata(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return nil
	}

	value := ctx.Value(MetadataKey)
	if value == nil {
		return nil
	}

	m, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	return m
}

// Set sets the value in the metadata
func Set(ctx context.Context, key string, value interface{}) bool {
	if key == "" {
		return false
	}

	meta := getMetadata(ctx)
	if meta == nil {
		return false
	}

	meta[key] = value
	return true
}

// Delete deletes the metadata
func Delete(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}

	meta := getMetadata(ctx)
	if meta == nil {
		return false
	}

	_, ok := meta[key]
	if !ok {
		return false
	}

	delete(meta, key)
	return true
}

// Read reads a value from the metadata
func Read(ctx context.Context, key string) (interface{}, bool) {
	if key == "" {
		return nil, false
	}

	meta := getMetadata(ctx)
	if meta == nil {
		return nil, false
	}

	val, ok := meta[key]
	return val, ok
}

// ReadString reads a string from the metadata
func ReadString(ctx context.Context, key string) (string, bool) {
	value, ok := Read(ctx, key)
	if !ok {
		return "", false
	}

	v, ok := value.(string)
	return v, ok
}

// ReadBool reads a boolean from the metadata
func ReadBool(ctx context.Context, key string) (bool, bool) {
	value, ok := Read(ctx, key)
	if !ok {
		return false, false
	}

	v, ok := value.(bool)
	return v, ok
}

// ReadInt reads an int from the metadata
func ReadInt(ctx context.Context, key string) (int, bool) {
	value, ok := Read(ctx, key)
	if !ok {
		return 0, false
	}

	v, ok := value.(int)
	return v, ok
}
