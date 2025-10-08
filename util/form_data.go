package util

import (
	"net/http"
)

// FormData is a convenient struct for passing user input and errors to HTML
// forms and showing to the user.
type FormData struct {
	err	   error
	fields map[string]string
}

// NewFormData constructs empty form data instance.
func NewFormData() FormData {
	return FormData{
		err:	nil,
		fields: map[string]string{},
	}
}

// FormDataFromRequest extracts a form data from request, using a set of keys.
func FormDataFromRequest(r *http.Request, keys []string) FormData {
	formData := NewFormData()
	for _, key := range keys {
		formData.Put(key, r.FormValue(key))
	}
	return formData
}

// HasError is true if there is indeed an error.
func (f FormData) HasError() bool {
	return f.err != nil
}

// Error returns an error text or empty string, if there are no errors in form data.
func (f FormData) Error() string {
	if f.err == nil {
		return ""
	}
	return f.err.Error()
}

// WithError puts an error into form data and returns itself.
func (f FormData) WithError(err error) FormData {
	f.err = err
	return f
}

// Get accesses form data with a key
func (f FormData) Get(key string) string {
	return f.fields[key]
}

// Put writes a form value for provided key
func (f FormData) Put(key, value string) {
	f.fields[key] = value
}
