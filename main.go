package main

import (
	"fmt"
	"os"
	"strings"
)

func STR_ENDS_WITH(s, part string) bool {
	return strings.HasSuffix(s, part)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: qoiconv <infile> <outfile>")
		fmt.Println("Examples:")
		fmt.Println("  qoiconv input.png output.qoi")
		fmt.Println("  qoiconv input.qoi output.png")
		return
	}

	/*
		//void *pixels = NULL;
		//int w, h, channels;
		if (STR_ENDS_WITH(argv[1], ".png")) {
			if(!stbi_info(argv[1], &w, &h, &channels)) {
				printf("Couldn't read header %s\n", argv[1]);
				exit(1);
			}

			// Force all odd encodings to be RGBA
			if(channels != 3) {
				channels = 4;
			}

			pixels = (void *)stbi_load(argv[1], &w, &h, NULL, channels);
		}
		else if (STR_ENDS_WITH(argv[1], ".qoi")) {
			qoi_desc desc;
			pixels = qoi_read(argv[1], &desc, 0);
			channels = desc.channels;
			w = desc.width;
			h = desc.height;
		}

		if (pixels == NULL) {
			printf("Couldn't load/decode %s\n", argv[1]);
			exit(1);
		}

		int encoded = 0;
		if (STR_ENDS_WITH(argv[2], ".png")) {
			encoded = stbi_write_png(argv[2], w, h, channels, pixels, 0);
		}
		else if (STR_ENDS_WITH(argv[2], ".qoi")) {
			encoded = qoi_write(argv[2], pixels, &(qoi_desc){
				.width = w,
				.height = h,
				.channels = channels,
				.colorspace = QOI_SRGB
			});
		}

		if (!encoded) {
			printf("Couldn't write/encode %s\n", argv[2]);
			exit(1);
		}

		free(pixels);
		return 0;
	*/
}
