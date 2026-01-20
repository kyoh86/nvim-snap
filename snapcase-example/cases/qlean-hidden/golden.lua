vim.fn.setline(1, { "foobar" })
vim.cmd.redraw()
require("nvim_snap").done()
