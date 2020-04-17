# It seems that the venv doesn't work in the makefile. https://stackoverflow.com/questions/44052093/makefile-with-source-get-error-no-such-file-or-directory
# If you want to use virtual environment, you can create the virtual environment first, then run the make file, here is the example
# python3 -m venv .venv
# source .venv/bin/activate

sdk_dev_install:
	pip install -e sdk/python
sdk_unittest:
	sdk/python/tests/run_tests.sh
sdk_kfp_sample_test:
	sdk/python/tests/test_kfp_samples.sh	
