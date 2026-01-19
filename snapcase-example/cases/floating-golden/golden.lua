vim.cmd("enew")
vim.fn.setline(1, {
  "Main Buffer",
  "",
  "Press ? for help",
  "Press q to quit",
})

local buf = vim.api.nvim_create_buf(false, true)
vim.api.nvim_buf_set_lines(buf, 0, -1, false, {
  "Palette",
  "--------",
  "alpha",
  "beta",
  "gamma",
})

vim.api.nvim_open_win(buf, false, {
  relative = "editor",
  row = 2,
  col = 4,
  width = 24,
  height = 5,
  style = "minimal",
  border = "rounded",
})

vim.cmd.redraw()
snap_done()
