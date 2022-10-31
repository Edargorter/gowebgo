#!/bin/bash -e

pre="[Gowebgo]"

echo -en "$pre Checking for go...\n"

echo -en "
  ______       _  _  _       _      ______       
 / _____)     | || || |     | |    / _____)      
| /  ___  ___ | || || | ____| | _ | /  ___  ___  
| | (___)/ _ \| ||_|| |/ _  ) || \| | (___)/ _ \ 
| \____/| |_| | |___| ( (/ /| |_) ) \____/| |_| |
 \_____/ \___/ \______|\____)____/ \_____/ \___/ 
 "
echo ""

#Check system software requirements 
if { [ "$(uname)" == 'Linux' ] && ! [ -x "$(command -v go)" ]; }; then
	echo "Linux"
	if [ $(which apt) ]; then
		echo "$pre Installing system requirements..."
		sudo apt install golang-go
	fi
elif [[ "$OSTYPE" == "darwin"* ]]; then
	if [ ! $(which brew) ]; then
		echo "$pre Installing brew..."
		/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
		echo "$pre Installing golang..."
		brew install go
	fi
fi

# Check directory 
if [ $(ls | grep gowebgo.go) == "gowebgo.go" ]; then
	echo "$pre Building gowebgo..."
	go build .
	echo "$pre Generating symlink..."
	if [[ "$(uname)" == "Linux"* ]]; then
		sudo ln -sf $PWD/gowebgo /usr/local/bin/gwg
	elif [[ "$(uname)" == "Darwin"* ]]; then
		sudo ln -sf $PWD/gowebgo /usr/local/bin/gwg
	fi
	echo "Installation complete. Go web go! Run with 'gwg'."
else
	echo "Please navigate to 'gowebgo' folder and restart installation."
fi
