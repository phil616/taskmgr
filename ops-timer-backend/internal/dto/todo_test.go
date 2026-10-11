package dto

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestTodoScoreValidation(t *testing.T) {
	for _, field := range []string{"importance", "difficulty"} {
		for _, tc := range []struct {
			value string
			valid bool
		}{
			{"0", true}, {"1", true}, {"0.375", true}, {"-0.001", false}, {"1.001", false}, {`"0.5"`, false}, {"true", false}, {"1e999", false},
		} {
			t.Run(field+"/"+tc.value, func(t *testing.T) {
				raw := []byte(fmt.Sprintf(`{"title":"test","%s":%s}`, field, tc.value))
				for _, req := range []any{&CreateTodoRequest{}, &UpdateTodoRequest{}} {
					err := json.Unmarshal(raw, req)
					if err == nil {
						err = binding.Validator.ValidateStruct(req)
					}
					if (err == nil) != tc.valid {
						t.Errorf("%T: valid=%v, error=%v", req, tc.valid, err)
					}
				}
			})
		}
	}
}
