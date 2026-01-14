local M = {}

function M.alloc_row(cols)
  local row = {}
  for c = 1, cols do
    row[c] = { text = " ", hl_id = 0 }
  end
  return row
end

function M.ensure_grid(grid, rows, cols)
  grid.rows = rows
  grid.cols = cols
  grid.cells = grid.cells or {}
  for r = 1, rows do
    local row = grid.cells[r]
    if not row then
      grid.cells[r] = M.alloc_row(cols)
    else
      if #row < cols then
        for c = #row + 1, cols do
          row[c] = { text = " ", hl_id = 0 }
        end
      elseif #row > cols then
        for c = cols + 1, #row do
          row[c] = nil
        end
      end
    end
  end
  for r = rows + 1, #grid.cells do
    grid.cells[r] = nil
  end
end

function M.clear_grid(grid)
  if not grid.rows or not grid.cols then
    return
  end
  for r = 1, grid.rows do
    local row = grid.cells[r]
    if not row then
      row = M.alloc_row(grid.cols)
      grid.cells[r] = row
    else
      for c = 1, grid.cols do
        row[c] = { text = " ", hl_id = 0 }
      end
    end
  end
end

function M.copy_cell(cell)
  return { text = cell.text, hl_id = cell.hl_id }
end

function M.scroll_grid(grid, top, bot, left, right, rows)
  if rows == 0 or not grid.cells then
    return
  end
  local top_r = top + 1
  local bot_r = bot
  local left_c = left + 1
  local right_c = right
  if rows > 0 then
    for r = top_r, bot_r - rows do
      local src = grid.cells[r + rows]
      local dst = grid.cells[r]
      for c = left_c, right_c do
        dst[c] = M.copy_cell(src[c])
      end
    end
    for r = bot_r - rows + 1, bot_r do
      local row = grid.cells[r]
      for c = left_c, right_c do
        row[c] = { text = " ", hl_id = 0 }
      end
    end
  else
    local offset = -rows
    for r = bot_r, top_r + offset, -1 do
      local src = grid.cells[r - offset]
      local dst = grid.cells[r]
      for c = left_c, right_c do
        dst[c] = M.copy_cell(src[c])
      end
    end
    for r = top_r, top_r + offset - 1 do
      local row = grid.cells[r]
      for c = left_c, right_c do
        row[c] = { text = " ", hl_id = 0 }
      end
    end
  end
end

return M
