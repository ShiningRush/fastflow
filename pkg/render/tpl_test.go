package render

import (
	"github.com/golang/groupcache/lru"
	"github.com/golang/mock/gomock"
	"sync"
	"testing"
	"text/template"
)

func TestCachedTplGetter_GetTpl(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	type fields struct {
		parseTplProvider TplProvider
		cache            *lru.Cache
	}
	type args struct {
		tplTexts []string
	}
	type TestCace struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		mockFn  func(tt TestCace)
	}
	tests := []TestCace{
		{
			name: "once succ",
			fields: fields{
				parseTplProvider: NewMockTplProvider(ctl),
				cache:            lru.New(10),
			},
			args: args{
				tplTexts: []string{
					"aaa",
				},
			},
			wantErr: false,
			mockFn: func(tt TestCace) {
				mockTplProvider := tt.fields.parseTplProvider.(*MockTplProvider)
				mockTplProvider.EXPECT().GetTpl(gomock.Any()).DoAndReturn(func(tplText string) (*template.Template, error) {
					return NewParseTplProvider().GetTpl(tplText)
				}).Times(1)
			},
		},
		{
			name: "once failed",
			fields: fields{
				parseTplProvider: NewMockTplProvider(ctl),
				cache:            lru.New(10),
			},
			args: args{
				tplTexts: []string{
					"{{a}}",
				},
			},
			wantErr: true,
			mockFn: func(tt TestCace) {
				mockTplProvider := tt.fields.parseTplProvider.(*MockTplProvider)
				mockTplProvider.EXPECT().GetTpl(gomock.Any()).DoAndReturn(func(tplText string) (*template.Template, error) {
					return NewParseTplProvider().GetTpl(tplText)
				}).Times(1)
			},
		},
		{
			name: "lru cache times",
			fields: fields{
				parseTplProvider: NewMockTplProvider(ctl),
				cache:            lru.New(3),
			},
			args: args{
				tplTexts: []string{
					"1", "1", "1", "2", "3", "4", "1",
				},
			},
			wantErr: false,
			mockFn: func(tt TestCace) {
				mockTplProvider := tt.fields.parseTplProvider.(*MockTplProvider)
				mockTplProvider.EXPECT().GetTpl(gomock.Any()).DoAndReturn(func(tplText string) (*template.Template, error) {
					return NewParseTplProvider().GetTpl(tplText)
				}).Times(5)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn(tt)
			c := &CachedTplProvider{
				parseTplProvider: tt.fields.parseTplProvider,
				cache:            tt.fields.cache,
				rwMutex:          sync.RWMutex{},
			}

			for i, text := range tt.args.tplTexts {
				_, err := c.GetTpl(text)
				if (err != nil) != tt.wantErr {
					t.Errorf("GetTpl() index:%d error = %v, wantErr %v", i, err, tt.wantErr)
					return
				}
			}

		})
	}
}
