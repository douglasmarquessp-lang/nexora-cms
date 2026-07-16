package example

import (
	"context"
	"fmt"
)

func AfterPostSaveHook(ctx context.Context, args map[string]interface{}) error {
	postID, _ := args["post_id"].(string)
	fmt.Printf("[Example Plugin] AfterPostSave hook fired for post %s\n", postID)
	return nil
}

func BeforeRenderHook(ctx context.Context, args map[string]interface{}) error {
	template, _ := args["template"].(string)
	fmt.Printf("[Example Plugin] BeforeRender hook fired for template %s\n", template)
	return nil
}

func AfterUploadHook(ctx context.Context, value interface{}, args map[string]interface{}) (interface{}, error) {
	filename, _ := args["filename"].(string)
	fmt.Printf("[Example Plugin] AfterUpload hook fired for file %s\n", filename)
	return value, nil
}
