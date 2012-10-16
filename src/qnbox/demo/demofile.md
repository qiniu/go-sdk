all:
	cd base; git pull
	cd bd; git pull
	cd enterprise; git pull
	cd fileop; git pull
	cd io; git pull
	cd service; git pull
	cd qboxrsp; git pull
	cd web; git pull

clean:
	@echo
