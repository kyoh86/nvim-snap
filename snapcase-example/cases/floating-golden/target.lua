vim.cmd("enew")
vim.fn.setline(1, {
  "Main Buffer",
  "(New message)",
  "Press ? for help",
  "Press q to quit",
})

local buf = vim.api.nvim_create_buf(false, true)
local lines = { "Palette", "--------", "one", "two", "three" }
vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)

vim.api.nvim_open_win(buf, false, {
  relative = "editor",
  row = 2,
  col = 4,
  width = 24,
  height = 5,
  style = "minimal",
  border = "rounded",
})
