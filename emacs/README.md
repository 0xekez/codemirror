# Emacs Installation

Getting set up using emacs is simple and easy. Here's how:

## Installation

1. Download the [mirror.el](mirror.el) file in this repo and place it
   somewhere you'll remember. For the purposes of this example, lets
   say that we've downloaded it to `~/.emacs.d/lisp/mirror.el`.

2. Add the installation directory to your emacs
   [`load-path`](https://www.emacswiki.org/emacs/LoadPath) by adding
   the following to your `init.el` or `.emacs` file:

   ```lisp
   (add-to-list 'load-path "~/.emacs.d/lisp")
   ```

## Usage

- Start mirroring: `M-x start-mirroring`
- Stop mirroring: `M-x stop-mirroring`