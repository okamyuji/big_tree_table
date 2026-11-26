# Loads every file under app/lib/ — shared modules and plain classes.
#
# First load group: nothing here may need a model at load time. The scan is
# recursive so subfolders load without editing this file.
# >>> snails generate
# scan: lib/**/*
require_relative "../app/lib/api_guard"
require_relative "../app/lib/money"
# <<< snails generate
