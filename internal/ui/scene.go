package ui

import (
	"context"
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/oakmound/oak/v4"
	"github.com/oakmound/oak/v4/dlog"
	"github.com/oakmound/oak/v4/event"
	"github.com/oakmound/oak/v4/key"
	"github.com/oakmound/oak/v4/render"
	"github.com/oakmound/oak/v4/scene"
	"github.com/paulmach/orb"

	"github.com/arsham/lify/internal/config"
	"github.com/arsham/lify/internal/entity/food"
)

const (
	sceneLoading = "loading_scene"
	sceneLify    = "lify_scene"
)

// Scene is a struct that represents a scene in the UI. It manages the
// transition between scenes and the rendering of the current scene.
type Scene struct {
	env   *config.Env
	board *Board
	win   *oak.Window
}

// NewScene creates a new Scene and sets up the drawing stack.
func NewScene(env *config.Env, b *Board) (*Scene, error) {
	render.SetDrawStack(
		render.NewCompositeR(),
		render.NewDynamicHeap(),
		render.NewStaticHeap(),
	)
	win := oak.NewWindow()
	s := &Scene{
		env:   env,
		board: b,
		win:   win,
	}

	err := win.AddScene(sceneLify, s.startLifyScene())
	if err != nil {
		return nil, fmt.Errorf("add main scene: %w", err)
	}

	err = s.win.AddScene(sceneLoading, s.loadingScene())
	if err != nil {
		return nil, fmt.Errorf("add loading scene: %w", err)
	}
	return s, nil
}

// Start starts the loading scene, and then transitions to the main scene.
func (s *Scene) Start() error {
	return s.win.Init(sceneLoading, func(c oak.Config) (oak.Config, error) {
		c.FrameRate = 60
		c.DrawFrameRate = 60
		c.Screen.Width = s.env.UI.Width
		c.Screen.Height = s.env.UI.Height
		c.Debug = oak.Debug{
			Level: "Info",
		}
		c.Title = "Lify Simulator"
		c.TrackInputChanges = true
		c.LoadBuiltinCommands = true
		c.TopMost = true
		c.BatchLoad = false
		c.Fullscreen = true
		return c, nil
	})
}

// loadingScene returns a scene that loads the assets in a goroutine and then
// transitions to the main scene. If any of the assets fail to load, it quits
// the game.
func (s *Scene) loadingScene() scene.Scene {
	return scene.Scene{
		Start: func(ctx *scene.Context) {
			titleText := render.NewText("Loading assets...", 0, 0)
			titleText.SetFont(s.board.Font(AssetFontInfo))
			putCentre(ctx, titleText, axixXY)
			_, err := render.Draw(titleText)
			if err != nil {
				dlog.Error("Failed rendering text:", err)
				ctx.Window.Quit()
				return
			}

			event.GlobalBind(ctx, key.Down(key.Q), func(key.Event) event.Response {
				ctx.Window.Quit()
				return 0
			})

			go func() {
				err := s.board.Load()
				if err != nil {
					dlog.Error("Failed loading assets:", err)
					ctx.Window.Quit()
					return
				}
				asset, err := s.board.Asset(AssetHerb1)
				if err != nil {
					dlog.Error("getting herb asset:", err)
					ctx.Window.Quit()
					return
				}

				bBounds := s.board.Bound()
				for i := 0; i < 10000; i++ {
					point := orb.Point{
						float64(rand.Int31n(int32(bBounds.Max.X()))),
						float64(rand.Int31n(int32(bBounds.Max.Y()))),
					}
					herb := food.NewHerb(fmt.Sprintf("Herb #%d", i))
					e := NewEntity(herb, point, asset)
					if err := s.board.Add(e); err != nil {
						dlog.Error("Failed adding at:", point)
					}
				}

				titleText.SetString("Assets have been loaded")
				titleText.SetFont(s.board.Font(AssetFontInfo))
				putCentre(ctx, titleText, axixXY)
				bounds := ctx.Window.Bounds()
				instructions := render.NewText("Press Enter to start, or press Q to quit", 0, float64(bounds.Y()*3/4))
				instructions.SetFont(s.board.Font(AssetFontInfo))
				putCentre(ctx, instructions, axixX)

				_, err = render.Draw(instructions)
				if err != nil {
					dlog.Error("Failed rendering text:", err)
					ctx.Window.Quit()
					return
				}

				event.GlobalBind(ctx, key.AnyDown, func(key.Event) event.Response {
					ctx.Window.NextScene()
					return 0
				})
			}()
		},
		End: func() (string, *scene.Result) {
			return sceneLify, nil
		},
	}
}

func (s *Scene) startLifyScene() scene.Scene {
	return scene.Scene{
		Start: func(ctx *scene.Context) {
			event.GlobalBind(ctx, key.Down(key.Q), func(key.Event) event.Response {
				ctx.Window.Quit()
				return 0
			})
			s.win.ParentContext = context.WithValue(context.Background(), preLoadTimeStr, time.Now())
			screen := render.NewColorBoxM(s.win.Bounds().X(), s.win.Bounds().Y(), color.RGBA{0, 0, 0, 0})

			for _, e := range s.board.EntitiesIn(2000, 300, 2000+1920, 300+1080) {
				e.Draw(screen, orb.Point{2000, 300})
			}
			_, err := render.Draw(screen)
			if err != nil {
				dlog.Error("Failed rendering text:", err)
				ctx.Window.Quit()
				return
			}
			s.win.SetLoadingRenderable(screen)
		},
		End: func() (string, *scene.Result) {
			return sceneLify, nil
		},
	}
}
