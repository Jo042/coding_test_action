package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Aicon-assignment/internal/domain/entity"
	domainErrors "Aicon-assignment/internal/domain/errors"
	"Aicon-assignment/internal/usecase"

	"github.com/labstack/echo/v4"
)

//
// モック用の ItemUsecase 実装
//

type mockItemUsecase struct {
	UpdateItemFunc func(ctx context.Context, id int64, input usecase.UpdateItemInput) (*entity.Item, error)
}

func (m *mockItemUsecase) GetAllItems(ctx context.Context) ([]*entity.Item, error) {
	return nil, nil
}

func (m *mockItemUsecase) GetItemByID(ctx context.Context, id int64) (*entity.Item, error) {
	return nil, nil
}

func (m *mockItemUsecase) CreateItem(ctx context.Context, input usecase.CreateItemInput) (*entity.Item, error) {
	return nil, nil
}

func (m *mockItemUsecase) DeleteItem(ctx context.Context, id int64) error {
	return nil
}

func (m *mockItemUsecase) GetCategorySummary(ctx context.Context) (*usecase.CategorySummary, error) {
	return &usecase.CategorySummary{}, nil
}

func (m *mockItemUsecase) UpdateItem(ctx context.Context, id int64, input usecase.UpdateItemInput) (*entity.Item, error) {
	if m.UpdateItemFunc != nil {
		return m.UpdateItemFunc(ctx, id, input)
	}
	return nil, nil
}

//
// テスト本体
//

// 正常系: name だけ更新できる
func TestUpdateItem_Success_NameOnly(t *testing.T) {
	e := echo.New()

	// リクエスト作成
	body := `{"name":"更新後の名前"}`
	req := httptest.NewRequest(http.MethodPatch, "/items/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Echo のコンテキスト作成
	c := e.NewContext(req, rec)
	c.SetPath("/items/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	// モック Usecase を用意
	mockUsecase := &mockItemUsecase{
		UpdateItemFunc: func(ctx context.Context, id int64, input usecase.UpdateItemInput) (*entity.Item, error) {
			if id != 1 {
				t.Fatalf("expected id=1, got=%d", id)
			}
			if input.Name == nil || *input.Name != "更新後の名前" {
				t.Fatalf("expected name=更新後の名前, got=%v", input.Name)
			}

			// 返却するエンティティ
			return &entity.Item{
				ID:            1,
				Name:          "更新後の名前",
				Category:      "時計",
				Brand:         "ROLEX",
				PurchasePrice: 1500000,
				PurchaseDate:  "2023-01-15",
			}, nil
		},
	}

	handler := NewItemHandler(mockUsecase)

	// ハンドラ実行
	if err := handler.UpdateItem(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	// ステータスコード検証
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// レスポンス JSON を構造体へ
	var resp entity.Item
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.ID != 1 {
		t.Errorf("expected id=1, got=%d", resp.ID)
	}
	if resp.Name != "更新後の名前" {
		t.Errorf("expected name=更新後の名前, got=%s", resp.Name)
	}
}

// 異常系: 1項目も指定されていない → 400
func TestUpdateItem_BadRequest_NoFields(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPatch, "/items/1", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.SetPath("/items/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	// Update は呼ばれない想定なので、空のモックでOK
	mockUsecase := &mockItemUsecase{}
	handler := NewItemHandler(mockUsecase)

	if err := handler.UpdateItem(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	// エラーメッセージの一部をチェック
	if !strings.Contains(rec.Body.String(), "validation failed") {
		t.Errorf("expected validation error message, got=%s", rec.Body.String())
	}
}

// 異常系: usecase 側でバリデーションエラー（負の価格など）→ 400
func TestUpdateItem_ValidationError_FromUsecase(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPatch, "/items/1", strings.NewReader(`{"purchase_price":-1}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.SetPath("/items/:id")
	c.SetParamNames("id")
	c.SetParamValues("1")

	mockUsecase := &mockItemUsecase{
		UpdateItemFunc: func(ctx context.Context, id int64, input usecase.UpdateItemInput) (*entity.Item, error) {
			// usecase から「バリデーションエラー」相当のエラーを返す想定
			return nil, domainErrors.ErrInvalidInput
		},
	}

	handler := NewItemHandler(mockUsecase)

	if err := handler.UpdateItem(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

// 異常系: 対象IDが存在しない → 404
func TestUpdateItem_NotFound(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPatch, "/items/999", strings.NewReader(`{"name":"hoge"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.SetPath("/items/:id")
	c.SetParamNames("id")
	c.SetParamValues("999")

	mockUsecase := &mockItemUsecase{
		UpdateItemFunc: func(ctx context.Context, id int64, input usecase.UpdateItemInput) (*entity.Item, error) {
			// usecase 側は「NotFound」を返した想定
			return nil, domainErrors.ErrItemNotFound
		},
	}

	handler := NewItemHandler(mockUsecase)

	if err := handler.UpdateItem(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

