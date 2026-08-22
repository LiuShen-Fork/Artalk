package meta_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/config/meta"
	"github.com/artalkjs/artalk/v2/test"
)

func Test_GetOptionsMetaData(t *testing.T) {
	app, _ := test.NewTestApp()
	defer app.Cleanup()

	result, err := meta.GetOptionsMetaData(config.Template("zh-CN"))
	if err != nil {
		t.Error(err)
	}

	if len(result) == 0 {
		t.Error("should get some metadata")
	}

	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}

func TestAIAssistantMetadata(t *testing.T) {
	app, _ := test.NewTestApp()
	defer app.Cleanup()

	result, err := meta.GetOptionsMetaData(config.Template("zh-CN"))
	if err != nil {
		t.Fatal(err)
	}

	metas := make(map[string]meta.OptionsMeta)
	for _, item := range result {
		if strings.HasPrefix(item.Path, "ai_assistant.") {
			metas[item.Path] = item
		}
	}

	if got := metas["ai_assistant.api_type"]; got.Title != "API 类型" {
		t.Fatalf("unexpected API type title: %q", got.Title)
	}
	if got := metas["ai_assistant.api_type"]; !reflect.DeepEqual(got.Options, []string{"responses", "chat_completions", "anthropic_messages"}) {
		t.Fatalf("unexpected API type options: %#v", got.Options)
	}
	if got := metas["ai_assistant.prompt"]; got.Title != "提示词" || got.Desc != "自定义助手行为" {
		t.Fatalf("unexpected prompt metadata: %#v", got)
	}
}
