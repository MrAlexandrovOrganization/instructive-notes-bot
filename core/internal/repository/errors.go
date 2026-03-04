package repository

import "errors"

// ErrNotFound is returned when an entity is not found in the database.
var ErrNotFound = errors.New("not found")
