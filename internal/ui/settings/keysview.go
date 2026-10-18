package settings

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/yangflow/idset/internal/account"
	"github.com/yangflow/idset/internal/keygen"
	"github.com/yangflow/idset/internal/ui/icons"
)

// keysPane is the form until a key is generated, then the key and what to do with it.
func (s *Screen) keysPane(c *ui.Context) {
	if s.keys.key != nil {
		s.keyResult(c)
		return
	}
	s.keyForm(c)
}

// keyLabel is the 168 DIP label at the start of a form row; files is the 96 DIP one of the Files group.
func keyLabel(c *ui.Context, text string, files bool) {
	t := c.Theme()
	w := t.Space(42)
	if files {
		w = t.Space(24)
	}
	ui.Text(c, text).Width(w).Shrink(0).SingleLine()
}

// textButton is a borderless button in the accent color, with an optional 12 DIP icon in front.
func textButton(c *ui.Context, key, label string, icon *ui.SVG) ui.Element {
	t := c.Theme()
	b := ui.Button(c.Key(key), "").Label(label).Background(ui.Color{}).Border(0, ui.Color{}).TextColor(t.Accent)
	b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(t.Space(3))
		}
		ui.Text(c, label).SingleLine()
	})
	return b
}

func (s *Screen) keyForm(c *ui.Context) {
	t := c.Theme()
	busy := s.keys.busy
	if a, ok := s.identity(s.keys.target); ok {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3), t.Space(2.5), t.Space(3)).Radius(10).Background(t.Surface).Children(func() {
			ui.Icon(c, icons.Key).FontSize(t.Space(3.5)).TextColor(t.TextMuted)
			ui.Text(c, fmt.Sprintf(s.text.KDiag, s.label(a), s.env.Tilde(a.KeyPath))).Grow(1).Wrap()
		})
	}
	names := []string{}
	for _, p := range account.Providers {
		names = append(names, s.providerName(p))
	}
	field := func(key string, value *string, label string, mono bool) {
		in := ui.TextInput(c.Key(key), value).Label(label).Grow(1).TextAlign(ui.End).Background(ui.Color{}).Border(0, ui.Color{}).Disabled(busy)
		if mono {
			in.Font("monospace").FontSize(t.Space(3))
		}
	}
	group := ui.Column(c).Gap(t.Space(5))
	if busy {
		group.Opacity(0.5)
	}
	group.Children(func() {
		s.section(c, s.text.KNew, func(r *rows) {
			r.line(func() {
				keyLabel(c, s.text.KType, false)
				ui.Spacer(c)
				ui.Segmented(c, &s.keys.kind, "Ed25519", "RSA 4096").Label(s.text.KType).Disabled(busy)
			})
			r.line(func() {
				keyLabel(c, s.text.KPlatform, false)
				ui.Spacer(c)
				ui.Select(c.Key("key-platform"), &s.keys.platform, names).Label(s.text.KPlatform).Disabled(busy)
			})
			r.line(func() {
				keyLabel(c, s.text.KFile, false)
				field("key-name", &s.keys.name, s.text.KFile, true)
			})
			r.line(func() {
				keyLabel(c, s.text.KComment, false)
				field("key-comment", &s.keys.comment, s.text.KComment, false)
			})
		}, func() {
			switch {
			case s.nameInvalid():
				s.warn(c, s.text.KInvalid)
			case s.keys.err != "" && s.keys.name == s.keys.made.Name:
				s.warn(c, s.keys.err)
			default:
				path := "~/.ssh/" + s.keys.name
				s.footnote(c, fmt.Sprintf(s.text.KFormNote, path, path+".pub"))
			}
		})
	})
	ui.Row(c).Gap(t.Space(2)).Justify(ui.End).AlignItems(ui.Center).Children(func() {
		if busy {
			ui.Text(c, s.text.Generating).FontSize(t.Space(2.75)).TextColor(t.TextMuted).SingleLine()
			ui.Spinner(c)
		}
		if ui.PrimaryButton(c.Key("generate"), s.text.Generate).Disabled(busy || s.nameInvalid()).Clicked() {
			s.generate()
		}
	})
}

func (s *Screen) keyResult(c *ui.Context) {
	t := c.Theme()
	key := *s.keys.key
	kind := "Ed25519"
	if s.keys.made.Type == keygen.RSA {
		kind = "RSA 4096"
	}
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Icon(c, icons.Check).FontSize(t.Space(4)).TextColor(t.Accent)
		ui.Text(c, s.text.Created).FontSize(t.Space(3.75)).FontWeight(600).Role(ui.RoleHeading).Level(2).SingleLine()
		ui.Text(c, kind+" · "+s.providerName(s.keys.provider)).FontSize(t.Space(2.75)).TextColor(t.TextMuted).SingleLine()
	})

	s.section(c, s.text.PubKey, func(r *rows) {
		if key.Public != "" {
			r.line(func() {
				ui.Text(c, key.Public).Font("monospace").FontSize(t.Space(3)).Grow(1).Wrap().Selectable()
			})
		}
		r.lineAt(t.Space(1), func() {
			flash := s.now().Before(s.keys.copied)
			if flash {
				c.After(s.keys.copied.Sub(s.now()))
				textButton(c, "copy-pub", s.text.Copied, icons.Check).Disabled(true)
				return
			}
			if textButton(c, "copy-pub", s.text.CopyPub, icons.Copy).Disabled(key.Public == "").Clicked() {
				c.WriteClipboard(key.Public)
				s.copyPublic()
			}
		})
	}, func() {
		if key.Public == "" {
			s.warn(c, fmt.Sprintf(s.text.NoPub, key.Path+".pub"))
		}
	})

	slot := func() { ui.Row(c).Size(t.Space(6), t.Space(6)).Shrink(0) }
	value := func(text string, mono bool) {
		v := ui.Text(c, text).Grow(1).TextAlign(ui.End).Wrap()
		if mono {
			v.Font("monospace").FontSize(t.Space(3))
		}
	}
	s.section(c, s.text.KFiles, func(r *rows) {
		r.line(func() {
			keyLabel(c, s.text.Priv, true)
			value(key.Path, true)
			if ghost(c, "reveal", s.revealLabel(), icons.Folder).Clicked() {
				s.act.Reveal(key.Path)
			}
		})
		r.line(func() {
			keyLabel(c, s.text.Pub, true)
			value(key.Path+".pub", true)
			slot()
		})
		if key.Fingerprint != "" {
			r.line(func() {
				keyLabel(c, s.text.Fp, true)
				value(key.Fingerprint, true)
				slot()
			})
		}
		r.line(func() {
			keyLabel(c, s.text.Pass, true)
			value(s.text.None, false)
			slot()
		})
	}, func() { s.footnote(c, fmt.Sprintf(s.text.PassNote, key.Path)) })

	p := s.keys.provider
	name := p.DisplayName()
	s.section(c, s.text.KNext, func(r *rows) {
		r.line(func() {
			if p == account.Custom || p.KeySettingsURL() == "" {
				ui.Text(c, s.text.AddCustom).Grow(1)
				ui.Text(c, s.text.AddCustomHint).FontSize(t.Space(2.75)).TextColor(t.TextMuted).SingleLine()
				return
			}
			ui.Text(c, fmt.Sprintf(s.text.AddOn, name)).Grow(1)
			label := fmt.Sprintf(s.text.OpenSettings, name)
			b := ui.Button(c.Key("open-settings"), "").Label(label)
			b.Children(func() {
				ui.Text(c, label).SingleLine()
				ui.Icon(c, icons.External).FontSize(t.Space(3))
			})
			if b.Clicked() {
				c.OpenURL(p.KeySettingsURL())
			}
		})
		r.line(func() {
			ui.Text(c, s.text.Agent).Grow(1)
			switch s.keys.agent {
			case agentLoaded:
				ui.Icon(c, icons.Check).FontSize(t.Space(3)).TextColor(t.Accent)
				ui.Text(c, s.text.Loaded).FontSize(t.Space(2.75)).TextColor(t.TextMuted).SingleLine()
			case agentBusy:
				ui.Spinner(c)
			default:
				if ui.Button(c.Key("load"), s.text.Load).Disabled(s.keys.agent == agentMissing).Clicked() {
					s.loadKey()
				}
			}
		})
		r.line(func() {
			ui.Text(c, s.text.UseFor).Grow(1)
			if s.keys.useBusy {
				ui.Spinner(c)
			}
			if a, ok := s.identity(s.keys.used); ok {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Shrink(0).Children(func() {
					dot(c, a.Color)
					ui.Text(c, s.label(a)).SingleLine()
				})
				if textButton(c, "change", s.text.Change, nil).Disabled(s.keys.useBusy || s.readOnly).Clicked() {
					s.openPick()
				}
				return
			}
			if ui.Button(c.Key("choose"), s.text.Choose).Disabled(s.keys.useBusy || s.readOnly).Clicked() {
				s.openPick()
			}
		})
	}, func() {
		quiet := true
		if s.keys.useErr != "" {
			s.warn(c, s.keys.useErr)
			quiet = false
		} else if a, ok := s.identity(s.keys.used); ok {
			s.note(c, icons.Check, t.Accent, t.TextMuted, fmt.Sprintf(s.text.UsedNote, s.label(a)))
			quiet = false
		}
		switch s.keys.agent {
		case agentMissing:
			s.note(c, icons.Warn, t.Warning, t.Warning, s.text.AgentFail)
			quiet = false
		case agentFailed:
			s.warn(c, s.keys.agentErr)
			quiet = false
		}
		switch {
		case !quiet:
		case p == account.Custom || p.KeySettingsURL() == "":
			s.footnote(c, s.text.NextCustom)
		default:
			s.footnote(c, fmt.Sprintf(s.text.NextNote, name))
		}
	})
	ui.Row(c).Justify(ui.End).Children(func() {
		if textButton(c, "another", s.text.Another, nil).Disabled(s.keys.useBusy || s.keys.agent == agentBusy).Clicked() {
			s.another()
		}
	})
}

// pickDialog asks which identity the new key is for: one of the list, or a new one made in the accounts window.
func (s *Screen) pickDialog(c *ui.Context) {
	if !s.keys.pickOpen || s.keys.key == nil {
		s.keys.pickOpen = false
		return
	}
	t := c.Theme()
	ui.DialogBase(c, &s.keys.pickOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(t.Space(90)).Padding(t.Space(5)).Gap(t.Space(3)).Radius(10).Background(t.Background).Border(1, t.Border).
			Role(ui.RoleDialog).Label(s.text.UseTitle)
		ui.Text(c, s.text.UseTitle).FontSize(t.Rem(1.15)).FontWeight(600).Role(ui.RoleHeading).Level(2).Wrap()
		ui.Text(c, s.text.UseBody).Wrap()
		ui.Divider(c)
		choice := func(value, label string, head func(), sub string) {
			on := s.keys.pick == value
			row := ui.RadioBase(c.Key("pick-"+value), &s.keys.pick, value).Label(label).Gap(t.Space(2)).Padding(t.Space(1.25), 0, t.Space(1.25), 0)
			row.Children(func() {
				ring := t.Border
				if on {
					ring = t.Accent
				}
				ui.Row(c).Size(t.Space(3.5), t.Space(3.5)).Radius(t.Space(1.75)).Border(1.5, ring).Justify(ui.Center).AlignItems(ui.Center).Shrink(0).
					Margin(t.Space(0.25), 0, 0, 0).Children(func() {
					if on {
						ui.Column(c).Size(t.Space(1.75), t.Space(1.75)).Radius(t.Space(0.875)).Background(t.Accent)
					}
				})
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(0.25)).Children(func() {
					ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(head)
					ui.Text(c, sub).FontSize(t.Space(2.75)).TextColor(t.TextMuted).Wrap()
				})
			})
		}
		ui.Column(c).Children(func() {
			for _, a := range s.accounts {
				sub := s.text.NoKey
				if a.KeyPath != "" {
					sub = fmt.Sprintf(s.text.NowKey, s.env.Tilde(a.KeyPath))
				}
				choice(a.ID, s.label(a), func() {
					dot(c, a.Color)
					ui.Text(c, s.label(a)).SingleLine().Shrink(0)
					ui.Text(c, a.GitUserEmail).FontSize(t.Space(2.75)).TextColor(t.TextMuted).SingleLine()
				}, sub)
			}
			sub := fmt.Sprintf(s.text.NewIDSubPlain, s.providerName(s.keys.provider))
			if email := s.draftEmail(); email != "" {
				sub = fmt.Sprintf(s.text.NewIDSub, email, s.providerName(s.keys.provider))
			}
			choice(newIdentity, s.text.NewID, func() { ui.Text(c, s.text.NewID).SingleLine() }, sub)
		})
		ui.Divider(c)
		ui.Text(c, s.text.UseNote).FontSize(t.Space(2.75)).TextColor(t.TextMuted).Wrap()
		ui.Row(c).Gap(t.Space(2)).Justify(ui.End).Children(func() {
			if ui.Button(c.Key("pick-cancel"), s.text.Cancel).AutoFocus().Clicked() {
				s.keys.pickOpen = false
			}
			if ui.PrimaryButton(c.Key("pick-use"), s.text.UseButton).Clicked() {
				s.confirmPick()
			}
		})
	})
}
