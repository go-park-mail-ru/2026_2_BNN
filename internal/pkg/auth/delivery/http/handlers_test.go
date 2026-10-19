package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bnn/internal/models"
	authDelivery "bnn/internal/pkg/auth/delivery/http"
	"bnn/internal/pkg/auth/delivery/http/api"
	"bnn/internal/pkg/auth/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSignInHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUC := mocks.NewMockUsecase(ctrl)
	handler := authDelivery.NewHandler(mockUC)

	t.Run("Success", func(t *testing.T) {
		reqBody := api.SignInJSONRequestBody{
			Login:    "testuser",
			Password: "password123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/signin", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		_ = rec

		mockUC.EXPECT().
			SignIn(gomock.Any(), "testuser", "password123").
			Return(models.User{Login: "testuser"}, models.AuthTokens{AccessToken: "access", RefreshToken: "refresh"}, nil)

		reqObj := api.SignInRequestObject{Body: &reqBody}

		resp, err := handler.SignIn(req.Context(), reqObj)

		require.NoError(t, err)
		assert.IsType(t, api.SignIn200JSONResponse{}, resp)
	})

	t.Run("Invalid Credentials", func(t *testing.T) {
		reqBody := api.SignInJSONRequestBody{
			Login:    "wrong",
			Password: "wrong",
		}

		mockUC.EXPECT().
			SignIn(gomock.Any(), "wrong", "wrong").
			Return(models.User{}, models.AuthTokens{}, models.ErrInvalidCredentials)

		reqObj := api.SignInRequestObject{Body: &reqBody}
		resp, err := handler.SignIn(context.Background(), reqObj)

		require.NoError(t, err)
		assert.IsType(t, api.SignIn401Response{}, resp)
	})
}
