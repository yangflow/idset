package accounts

import (
	"errors"
	"testing"

	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keyfile"
)

// TestMockupStates walks the 18 states of design/screens/accounts.html, in both languages and both schemes, and
// checks what each one says. The interactions behind them have their own tests.
func TestMockupStates(t *testing.T) {
	type texts struct{ en, zh []string }
	states := []struct {
		name  string
		list  []account.Account
		req   Request
		setup func(t *testing.T, r *rig)
		texts texts
	}{
		{"empty", []account.Account{}, Request{}, nil, texts{
			[]string{"No identities yet", "New"}, []string{"还没有身份", "新建"}}},
		{"global", sample(), Request{ID: personalID}, nil, texts{
			[]string{"Global", "Used when no folder rule matches.", "Last used"}, []string{"全局", "没有 gitdir 规则命中时生效。", "最近使用"}}},
		{"scoped", sample(), Request{ID: workID}, nil, texts{
			[]string{"Scoped", "Add Path", "Remove this path", "Push only"}, []string{"限定", "添加路径", "删除这条路径", "仅推送"}}},
		{"custom", sample(), Request{ID: clientID}, nil, texts{
			[]string{"Custom", "Host name", "Port"}, []string{"自定义", "主机名", "端口"}}},
		{"dirty", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.typeInto(t, r.s.text.Username, "bob-new")
		}, texts{[]string{"Unsaved changes", "Revert", "Save"}, []string{"有未保存的修改", "还原", "保存"}}},
		{"draft", sample(), Request{New: true}, func(t *testing.T, r *rig) {
			r.s.SetPrefill(r.s.Draft(), "Bob", "bob@example.com")
		}, texts{[]string{"New", "New identity, not saved yet", "From git config --global"},
			[]string{"新建", "新身份，还没有保存", "来自 git config --global"}}},
		{"rewrites", sample(), Request{ID: workID}, nil, texts{
			[]string{"URL rewrites", "Add Rewrite", "Add Platform Default Rewrite"}, []string{"地址改写", "添加改写", "添加平台默认改写"}}},
		{"invalid", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.click(t, r.s.text.ScopeGlobal)
			r.click(t, r.s.text.Save)
		}, texts{[]string{"1 problem to fix first", "There is already a global identity. Change personal to limited first."},
			[]string{"有 1 处需要先改正", "已有全局身份。先把 personal 改成限定。"}}},
		{"saving", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.typeInto(t, r.s.text.Username, "bob-new")
			r.click(t, r.s.text.Save)
			r.tt.Frame()
		}, texts{[]string{"Writing…", "Saving"}, []string{"正在写入…", "保存中"}}},
		{"saved", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.typeInto(t, r.s.text.Username, "bob-new")
			r.click(t, r.s.text.Save)
			r.saveSucceeds(t, nil)
		}, texts{[]string{"Written"}, []string{"已写入"}}},
		{"failed", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.typeInto(t, r.s.text.Username, "bob-new")
			r.click(t, r.s.text.Save)
			r.saveSucceeds(t, errors.New("permission denied"))
		}, texts{[]string{"Could not write ~/.config/idset/accounts.json: permission denied", "Save"},
			[]string{"无法写入 ~/.config/idset/accounts.json：permission denied", "保存"}}},
		{"del", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.None, "")
		}, texts{[]string{"Delete identity work?", "~/.ssh/id_ed25519_work", "~/clients/acme/", "Also delete the private key", "Cancel"},
			[]string{"删除身份 work？", "~/.ssh/id_ed25519_work", "~/clients/acme/", "同时删除私钥", "取消"}}},
		{"delon", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.None, "")
			r.click(t, r.s.text.AlsoKey)
			r.tt.Frame()
		}, texts{[]string{"Only the private key is deleted; the .pub file stays."}, []string{"只删私钥，保留 .pub 文件。"}}},
		{"delnokey", sample(), Request{ID: miscID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.NoPath, "")
		}, texts{[]string{"This identity has no key path.", "none"}, []string{"这个身份没有私钥路径。", "没有"}}},
		{"delshared", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.Shared, "personal")
		}, texts{[]string{"Another identity (personal) uses the same key."}, []string{"其他身份（personal）也在用这个私钥。"}}},
		{"delsymlink", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.Symlink, "")
		}, texts{[]string{"The key path is a symbolic link; Idset will not delete it."}, []string{"私钥路径是符号链接，Idset 不会删除。"}}},
		{"delmissing", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.Missing, "")
		}, texts{[]string{"The key file does not exist."}, []string{"私钥文件不存在。"}}},
		{"delnotfile", sample(), Request{ID: workID}, func(t *testing.T, r *rig) {
			r.askDelete(t, keyfile.NotFile, "")
		}, texts{[]string{"The key path is not a regular file."}, []string{"私钥路径不是普通文件。"}}},
	}
	for _, st := range states {
		for _, lang := range []string{"en", "zh-Hans"} {
			for _, dark := range []bool{false, true} {
				name := st.name + "/" + lang
				if dark {
					name += "/dark"
				}
				t.Run(name, func(t *testing.T) {
					r := open(t, lang, st.list, st.req)
					r.tt.SetDark(dark)
					if st.setup != nil {
						st.setup(t, r)
					}
					r.tt.Frame()
					want := st.texts.en
					if lang == "zh-Hans" {
						want = st.texts.zh
					}
					r.want(t, want...)
				})
			}
		}
	}
	if len(states) != 18 {
		t.Errorf("the mockup has 18 states, the test %d", len(states))
	}
}
