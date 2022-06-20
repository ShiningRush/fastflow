package render

import "testing"

func TestTplRender_Render(t1 *testing.T) {
	type fields struct {
		tplProvider TplProvider
	}
	type args struct {
		tplText string
		data    interface{}
		opts    []TplOption
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "parse fail",
			fields: fields{
				tplProvider: NewParseTplProvider(),
			},
			args: args{
				tplText: "{{a}}",
				data:    nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "succ",
			fields: fields{
				tplProvider: NewParseTplProvider(),
			},
			args: args{
				tplText: "{{.a.b.c}}",
				data: map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{
							"c": 12345,
						},
					},
				},
			},
			want:    "12345",
			wantErr: false,
		},
		{
			name: "no value",
			fields: fields{
				tplProvider: NewParseTplProvider(),
			},
			args: args{
				tplText: "{{.a.c}}",
				data:    map[string]interface{}{},
			},
			want:    "<no value>",
			wantErr: false,
		},
		{
			name: "MissKeyStrategyError",
			fields: fields{
				tplProvider: NewParseTplProvider(WithMissKeyStrategy(MissKeyStrategyError)),
			},
			args: args{
				tplText: "{{.a.c}}",
				data:    map[string]interface{}{},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "MissKeyStrategyError",
			fields: fields{
				tplProvider: NewParseTplProvider(WithMissKeyStrategy(MissKeyStrategyDefault)),
			},
			args: args{
				tplText: "{{.a.c}}",
				data:    map[string]interface{}{},
			},
			want:    "<no value>",
			wantErr: false,
		},
		{
			name: "MissKeyStrategyError",
			fields: fields{
				tplProvider: NewParseTplProvider(WithMissKeyStrategy(MissKeyStrategyInvalid)),
			},
			args: args{
				tplText: "{{.a.c}}",
				data:    map[string]interface{}{},
			},
			want:    "<no value>",
			wantErr: false,
		},
		{
			name: "MissKeyStrategyError",
			fields: fields{
				tplProvider: NewParseTplProvider(WithMissKeyStrategy(MissKeyStrategyZero)),
			},
			args: args{
				tplText: "{{.a}}",
				data:    map[string]interface{}{},
			},
			want:    "<no value>",
			wantErr: false,
		},
		{
			name: "MissKeyStrategyError string",
			fields: fields{
				tplProvider: NewParseTplProvider(WithMissKeyStrategy(MissKeyStrategyZero)),
			},
			args: args{
				tplText: "{{.a}}",
				data:    map[string]string{},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "MissKeyStrategyError int",
			fields: fields{
				tplProvider: NewParseTplProvider(WithMissKeyStrategy(MissKeyStrategyZero)),
			},
			args: args{
				tplText: "{{.a}}",
				data:    map[string]int{},
			},
			want:    "0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TplRender{
				tplProvider: tt.fields.tplProvider,
			}
			got, err := t.Render(tt.args.tplText, tt.args.data)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("Render() got = %v, want %v", got, tt.want)
			}
		})
	}
}
