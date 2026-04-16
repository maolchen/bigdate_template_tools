package server

import "testing"

func TestExtractInlineSkillIDs(t *testing.T) {
	ids := extractInlineSkillIDs("please use #skill:core/template_contract and @skill(task/install_script, custom/es_guard) plus #template")
	if len(ids) != 4 {
		t.Fatalf("unexpected skill id count: %d (%#v)", len(ids), ids)
	}
	expect := map[string]bool{
		"core/template_contract": true,
		"task/install_script":    true,
		"custom/es_guard":        true,
		"template":               true,
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
		"task/install_script",
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
		"task/install_script",
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

func TestDetectTemplateIntentDoesNotTriggerOnServiceNameOnly(t *testing.T) {
	ok, _ := detectTemplateIntent("elasticsearch 是什么组件", nil, []string{"elasticsearch"})
	if ok {
		t.Fatal("expected template intent to stay false for service-name-only question")
	}
}

func TestSelectPromptSkillsSkipsTemplateSkillsForGeneralQuestion(t *testing.T) {
	server := newTestServer(t)
	skills, err := server.selectPromptSkills(&aiSession{}, aiSessionMessageRequest{
		Message: "你是什么模型",
	}, nil, nil)
	if err != nil {
		t.Fatalf("selectPromptSkills error: %v", err)
	}
	if len(skills) != 0 {
		t.Fatalf("expected no skills for general question, got %#v", skills)
	}
}

func TestSelectPromptSkillsSupportsTemplateEntryAlias(t *testing.T) {
	server := newTestServer(t)
	skills, err := server.selectPromptSkills(&aiSession{}, aiSessionMessageRequest{
		Message:          "请进入模板模式",
		SelectedSkillIDs: []string{"template"},
	}, nil, nil)
	if err != nil {
		t.Fatalf("selectPromptSkills error: %v", err)
	}
	if len(skills) == 0 {
		t.Fatal("expected template entry alias to load template skills")
	}

	refSet := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		refSet[skill.Ref.ID] = struct{}{}
	}
	required := []string{
		"core/base",
		"core/output_json",
		"core/path_guard",
		"core/template_contract",
		"core/variable_contract",
		"core/template_functions",
	}
	for _, ref := range required {
		if _, ok := refSet[ref]; !ok {
			t.Fatalf("expected core skill %s in selection, got %#v", ref, refSet)
		}
	}
}
