local M = {}

local function rgb_to_ansi(color, is_bg)
  if color == nil or color == vim.NIL then
    return nil
  end
  if type(color) ~= "number" or color < 0 then
    return nil
  end
  local r = math.floor(color / 65536) % 256
  local g = math.floor(color / 256) % 256
  local b = color % 256
  return string.format("\x1b[%d;2;%d;%d;%dm", is_bg and 48 or 38, r, g, b)
end

function M.render_ansi(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return ""
  end

  local default_fg = snapshot.default_colors and snapshot.default_colors.rgb_fg or nil
  local default_bg = snapshot.default_colors and snapshot.default_colors.rgb_bg or nil

  local attr_map = {}
  for _, attr in ipairs(snapshot.hl_attrs or {}) do
    attr_map[attr.id] = attr.rgb_attr or {}
  end

  local function to_style(hl_id)
    local attr = attr_map[hl_id] or {}
    local fg = attr.foreground
    local bg = attr.background
    local reverse = attr.reverse == true
    if fg == nil then
      fg = default_fg
    end
    if bg == nil then
      bg = default_bg
    end
    if reverse then
      fg, bg = bg, fg
    end
    return {
      fg = fg,
      bg = bg,
      bold = attr.bold == true,
      italic = attr.italic == true,
      underline = attr.underline == true
        or attr.undercurl == true
        or attr.underdouble == true
        or attr.underdotted == true
        or attr.underdashed == true,
      strikethrough = attr.strikethrough == true,
      reverse = reverse,
    }
  end

  local function style_equal(a, b)
    return a.fg == b.fg
      and a.bg == b.bg
      and a.bold == b.bold
      and a.italic == b.italic
      and a.underline == b.underline
      and a.strikethrough == b.strikethrough
      and a.reverse == b.reverse
  end

  local function style_to_ansi(style)
    local codes = { "\x1b[0m" }
    if style.bold then
      table.insert(codes, "\x1b[1m")
    end
    if style.italic then
      table.insert(codes, "\x1b[3m")
    end
    if style.underline then
      table.insert(codes, "\x1b[4m")
    end
    if style.strikethrough then
      table.insert(codes, "\x1b[9m")
    end
    local fg = rgb_to_ansi(style.fg, false)
    local bg = rgb_to_ansi(style.bg, true)
    if fg then
      table.insert(codes, fg)
    end
    if bg then
      table.insert(codes, bg)
    end
    return table.concat(codes)
  end

  local out = {}
  for r = 1, grid.rows do
    local row_cells = grid.cells[r] or {}
    local current = {
      fg = nil,
      bg = nil,
      bold = false,
      italic = false,
      underline = false,
      strikethrough = false,
      reverse = false,
    }
    local line = {}
    for c = 1, grid.cols do
      local cell = row_cells[c] or { text = " ", hl_id = 0 }
      local text = cell.text
      if text == "" then
        text = " "
      end
      local style = to_style(cell.hl_id or 0)
      if not style_equal(style, current) then
        table.insert(line, style_to_ansi(style))
        current = style
      end
      table.insert(line, text)
    end
    table.insert(line, "\x1b[0m")
    table.insert(out, table.concat(line))
  end
  return table.concat(out, "\n")
end

local function to_hex_color(color)
  if color == nil or color == vim.NIL then
    return nil
  end
  if type(color) ~= "number" or color < 0 then
    return nil
  end
  return string.format("#%06x", color)
end

local function escape_html(text)
  return (text:gsub("[&<>\"']", {
    ["&"] = "&amp;",
    ["<"] = "&lt;",
    [">"] = "&gt;",
    ['"'] = "&quot;",
    ["'"] = "&#39;",
  }))
end

function M.render_html(snapshot)
  local grid = nil
  for _, g in ipairs(snapshot.grids or {}) do
    if g.id == 1 then
      grid = g
      break
    end
  end
  if not grid and snapshot.grids and snapshot.grids[1] then
    grid = snapshot.grids[1]
  end
  if not grid then
    return ""
  end

  local default_fg = snapshot.default_colors and snapshot.default_colors.rgb_fg or nil
  local default_bg = snapshot.default_colors and snapshot.default_colors.rgb_bg or nil

  local attr_map = {}
  for _, attr in ipairs(snapshot.hl_attrs or {}) do
    attr_map[attr.id] = attr.rgb_attr or {}
  end

  local function to_style(hl_id)
    local attr = attr_map[hl_id] or {}
    local fg = attr.foreground
    local bg = attr.background
    local reverse = attr.reverse == true
    if fg == nil then
      fg = default_fg
    end
    if bg == nil then
      bg = default_bg
    end
    if reverse then
      fg, bg = bg, fg
    end
    return {
      fg = fg,
      bg = bg,
      bold = attr.bold == true,
      italic = attr.italic == true,
      underline = attr.underline == true
        or attr.undercurl == true
        or attr.underdouble == true
        or attr.underdotted == true
        or attr.underdashed == true,
      strikethrough = attr.strikethrough == true,
    }
  end

  local function style_equal(a, b)
    return a.fg == b.fg
      and a.bg == b.bg
      and a.bold == b.bold
      and a.italic == b.italic
      and a.underline == b.underline
      and a.strikethrough == b.strikethrough
  end

  local function style_to_css(style)
    local parts = {}
    local fg = to_hex_color(style.fg)
    local bg = to_hex_color(style.bg)
    if fg then
      table.insert(parts, "color:" .. fg)
    end
    if bg then
      table.insert(parts, "background-color:" .. bg)
    end
    if style.bold then
      table.insert(parts, "font-weight:700")
    end
    if style.italic then
      table.insert(parts, "font-style:italic")
    end
    local decorations = {}
    if style.underline then
      table.insert(decorations, "underline")
    end
    if style.strikethrough then
      table.insert(decorations, "line-through")
    end
    if #decorations > 0 then
      table.insert(parts, "text-decoration:" .. table.concat(decorations, " "))
    end
    return table.concat(parts, ";")
  end

  local lines = {}
  for r = 1, grid.rows do
    local row_cells = grid.cells[r] or {}
    local current = {
      fg = nil,
      bg = nil,
      bold = false,
      italic = false,
      underline = false,
      strikethrough = false,
    }
    local line = {}
    local chunk = {}
    for c = 1, grid.cols do
      local cell = row_cells[c] or { text = " ", hl_id = 0 }
      local text = cell.text
      if text == "" then
        text = " "
      end
      local style = to_style(cell.hl_id or 0)
      if not style_equal(style, current) then
        if #chunk > 0 then
          local css = style_to_css(current)
          if css ~= "" then
            table.insert(line, '<span style="' .. css .. '">' .. escape_html(table.concat(chunk)) .. "</span>")
          else
            table.insert(line, escape_html(table.concat(chunk)))
          end
          chunk = {}
        end
        current = style
      end
      table.insert(chunk, text)
    end
    if #chunk > 0 then
      local css = style_to_css(current)
      if css ~= "" then
        table.insert(line, '<span style="' .. css .. '">' .. escape_html(table.concat(chunk)) .. "</span>")
      else
        table.insert(line, escape_html(table.concat(chunk)))
      end
    end
    table.insert(lines, table.concat(line))
  end

  local bg = to_hex_color(default_bg) or "#000000"
  local fg = to_hex_color(default_fg) or "#ffffff"
  return table.concat({
    "<!doctype html>",
    "<html>",
    "<head>",
    '  <meta charset="utf-8" />',
    "  <title>Neovim UI Snapshot</title>",
    "  <style>",
    "    body {",
    "      margin: 0;",
    "      background: " .. bg .. ";",
    "      color: " .. fg .. ";",
    "      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;",
    "      line-height: 1.2;",
    "    }",
    "    pre {",
    "      margin: 0;",
    "      padding: 16px;",
    "      white-space: pre;",
    "    }",
    "  </style>",
    "</head>",
    "<body>",
    "  <pre>" .. table.concat(lines, "\n") .. "</pre>",
    "</body>",
    "</html>",
  }, "\n")
end

return M
