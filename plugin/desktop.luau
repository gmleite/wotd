--!nonstrict
-- Desktop widget: a small dictionary entry.
--
--   word – translation (or definition)
--   • example sentence
--     its translation
--
-- The host already draws the tile background, so we don't draw our own
-- (an extra fill here looked like a "window inside a window").
-- Desktop widgets don't show tooltips, so everything is drawn on the tile.
-- Click the tile to force a refresh.
local core = require("./lib/core.luau")

local current = { word = "…", meaning = "", examples = {}, err = nil }

local function render()
  local children = {}

  if current.err then
    table.insert(children, ui.label({ text = "wotd", fontSize = 22, fontWeight = "bold", color = "on_surface" }))
    table.insert(children, ui.label({
      text = current.err, fontSize = 12, color = "on_surface", maxWidth = 320, maxLines = 4,
    }))
  else
    -- headword + meaning on one line
    table.insert(children, ui.row({ gap = 8, align = "center" }, {
      ui.label({ text = current.word, fontSize = 28, fontWeight = "bold", color = "on_surface" }),
      ui.label({
        text = "– " .. current.meaning,
        fontSize = 15,
        color = "primary",
        maxWidth = 260,
        maxLines = 3,
      }),
    }))

    -- examples
    for _, ex in ipairs(current.examples or {}) do
      table.insert(children, ui.label({
        text = "• " .. ex.text, fontSize = 13, color = "on_surface", maxWidth = 360, maxLines = 2,
      }))
      if ex.translation and ex.translation ~= "" then
        table.insert(children, ui.label({
          text = "   " .. ex.translation, fontSize = 12, color = "primary", maxWidth = 360, maxLines = 2,
        }))
      end
    end
  end

  desktopWidget.render(ui.column({ gap = 6, padding = 4, align = "start", onClick = "onTap" }, children))
end

local function refresh(force)
  core.fetch(force, function(s)
    current = s
    render()
  end)
end

-- Render and set the interval inside update(), as the docs do. The first
-- update() runs right after load; the helper caches per day, so hourly is plenty.
function update()
  noctalia.setUpdateInterval(3600 * 1000)
  render()
  refresh(false)
end

function onTap()
  refresh(true)
end
