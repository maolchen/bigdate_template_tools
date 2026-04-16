package server

import "testing"

func TestExtractInlineSkillIDs(t *testing.T) {
	ids := extractInlineSkillIDs("请按 #skill:core/template_contract 执行，并 @skill(service/elasticsearch, custom/es_guard)")
	if len(ids) != 3 {
		t.Fatalf("unexpected skill id count: %d (%#v)", len(ids), ids)
	}
	expect := map[string]bool{
		"core/template_contract": true,
		"service/elasticsearch":  true,
		"custom/es_guard":        true,
	}
	for _, id := range ids {
		if !expect[id] {
			t.Fatalf("unexpected skill id: %s", id)
		}
	}
}

func TestSelectExplicitPromptSkillsSupportsBuiltinAndCustom(t *testing.T) {
	server := newTestServer(t)
	if _, err := server.saveCustomSkill(aiCustomSkillUpsertRequest{
		ID:      "es_guard",
		Title:   "ES Guard",
		Scope:   "task",
		Content: "do not use NodeAlias",
	}); err != nil {
		t.Fatalf("saveCustomSkill error: %v", err)
	}

	selected, err := server.selectExplicitPromptSkills([]string{
		"core/template_contract",
		"service/elasticsearch",
		"custom/es_guard",
	})
	if err != nil {
		t.Fatalf("selectExplicitPromptSkills error: %v", err)
	}

	refSet := make(map[string]struct{}, len(selected))
	for _, skill := range selected {
		refSet[skill.Ref.ID] = struct{}{}
	}

	requiredRefs := []string{
		"core/template_contract",
		"service/elasticsearch",
		"custom/es_guard",
	}
	for _, ref := range requiredRefs {
		if _, ok := refSet[ref]; !ok {
			t.Fatalf("expected explicit skill %s in selection, got %#v", ref, refSet)
		}
	}
}

func TestDetectTemplateIntentByKeywordAndService(t *testing.T) {
	ok, reasons := detectTemplateIntent("请生成模板并修改 elasticsearch", nil, []string{"elasticsearch", "mysql"})
	if !ok {
		t.Fatal("expected template intent to be true")
	}
	if len(reasons) == 0 {
		t.Fatal("expected non-empty reasons")
	}
}
