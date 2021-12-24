/*

QOI - The "Quite OK Image" format for fast, lossless image compression

Dominic Szablewski - https://phoboslab.org


-- LICENSE: The MIT License(MIT)

Copyright(c) 2021 Dominic Szablewski

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files(the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and / or sell copies
of the Software, and to permit persons to whom the Software is furnished to do
so, subject to the following conditions :
The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.


-- About

QOI encodes and decodes images in a lossless format. Compared to stb_image and
stb_image_write QOI offers 20x-50x faster encoding, 3x-4x faster decoding and
20% better compression.


-- Synopsis

// Define `QOI_IMPLEMENTATION` in *one* C/C++ file before including this
// library to create the implementation.

 QOI_IMPLEMENTATION
#include "qoi.h"

// Encode and store an RGBA buffer to the file system. The qoi_desc describes
// the input pixel data.
qoi_write("image_new.qoi", rgba_pixels, &(qoi_desc){
	.width = 1920,
	.height = 1080,
	.channels = 4,
	.colorspace = QOI_SRGB
});

// Load and decode a QOI image from the file system into a 32bbp RGBA buffer.
// The qoi_desc struct will be filled with the width, height, number of channels
// and colorspace read from the file header.
qoi_desc desc;
void *rgba_pixels = qoi_read("image.qoi", &desc, 4);



-- Documentation

This library provides the following functions;
- qoi_read    -- read and decode a QOI file
- qoi_decode  -- decode the raw bytes of a QOI image from memory
- qoi_write   -- encode and write a QOI file
- qoi_encode  -- encode an rgba buffer into a QOI image in memory

See the function declaration below for the signature and more information.

If you don't want/need the qoi_read and qoi_write functions, you can define
QOI_NO_STDIO before including this library.

This library uses malloc() and free(). To supply your own malloc implementation
you can define QOI_MALLOC and QOI_FREE before including this library.

This library uses memset() to zero-initialize the index. To supply your own
implementation you can define QOI_ZEROARR before including this library.


-- Data Format

A QOI file has a 14 byte header, followed by any number of data "chunks" and an
8-byte end marker.

struct qoi_header_t {
	char     magic[4];   // magic bytes "qoif"
	uint32_t width;      // image width in pixels (BE)
	uint32_t height;     // image height in pixels (BE)
	uint8_t  channels;   // 3 = RGB, 4 = RGBA
	uint8_t  colorspace; // 0 = sRGB with linear alpha, 1 = all channels linear
};

Images are encoded row by row, left to right, top to bottom. The decoder and
encoder start with {r: 0, g: 0, b: 0, a: 255} as the previous pixel value. An
image is complete when all pixels specified by width * height have been covered.

Pixels are encoded as
 - a run of the previous pixel
 - an index into an array of previously seen pixels
 - a difference to the previous pixel value in r,g,b
 - full r,g,b or r,g,b,a values

The color channels are assumed to not be premultiplied with the alpha channel
("un-premultiplied alpha").

A running array[64] (zero-initialized) of previously seen pixel values is
maintained by the encoder and decoder. Each pixel that is seen by the encoder
and decoder is put into this array at the position formed by a hash function of
the color value. In the encoder, if the pixel value at the index matches the
current pixel, this index position is written to the stream as QOI_OP_INDEX.
The hash function for the index is:

	index_position = (r * 3 + g * 5 + b * 7 + a * 11) % 64

Each chunk starts with a 2- or 8-bit tag, followed by a number of data bits. The
bit length of chunks is divisible by 8 - i.e. all chunks are byte aligned. All
values encoded in these data bits have the most significant bit on the left.

The 8-bit tags have precedence over the 2-bit tags. A decoder must check for the
presence of an 8-bit tag first.

The byte stream's end is marked with 7 0x00 bytes followed a single 0x01 byte.


The possible chunks are:


.- QOI_OP_INDEX ----------.
|         Byte[0]         |
|  7  6  5  4  3  2  1  0 |
|-------+-----------------|
|  0  0 |     index       |
`-------------------------`
2-bit tag b00
6-bit index into the color index array: 0..63

A valid encoder must not issue 7 or more consecutive QOI_OP_INDEX chunks to the
index 0, to avoid confusion with the 8 byte end marker.


.- QOI_OP_DIFF -----------.
|         Byte[0]         |
|  7  6  5  4  3  2  1  0 |
|-------+-----+-----+-----|
|  0  1 |  dr |  dg |  db |
`-------------------------`
2-bit tag b01
2-bit   red channel difference from the previous pixel between -2..1
2-bit green channel difference from the previous pixel between -2..1
2-bit  blue channel difference from the previous pixel between -2..1

The difference to the current channel values are using a wraparound operation,
so "1 - 2" will result in 255, while "255 + 1" will result in 0.

Values are stored as unsigned integers with a bias of 2. E.g. -2 is stored as
0 (b00). 1 is stored as 3 (b11).

The alpha value remains unchanged from the previous pixel.


.- QOI_OP_LUMA -------------------------------------.
|         Byte[0]         |         Byte[1]         |
|  7  6  5  4  3  2  1  0 |  7  6  5  4  3  2  1  0 |
|-------+-----------------+-------------+-----------|
|  1  0 |  green diff     |   dr - dg   |  db - dg  |
`---------------------------------------------------`
2-bit tag b10
6-bit green channel difference from the previous pixel -32..31
4-bit   red channel difference minus green channel difference -8..7
4-bit  blue channel difference minus green channel difference -8..7

The green channel is used to indicate the general direction of change and is
encoded in 6 bits. The red and blue channels (dr and db) base their diffs off
of the green channel difference and are encoded in 4 bits. I.e.:
	dr_dg = (last_px.r - cur_px.r) - (last_px.g - cur_px.g)
	db_dg = (last_px.b - cur_px.b) - (last_px.g - cur_px.g)

The difference to the current channel values are using a wraparound operation,
so "10 - 13" will result in 253, while "250 + 7" will result in 1.

Values are stored as unsigned integers with a bias of 32 for the green channel
and a bias of 8 for the red and blue channel.

The alpha value remains unchanged from the previous pixel.


.- QOI_OP_RUN ------------.
|         Byte[0]         |
|  7  6  5  4  3  2  1  0 |
|-------+-----------------|
|  1  1 |       run       |
`-------------------------`
2-bit tag b11
6-bit run-length repeating the previous pixel: 1..62

The run-length is stored with a bias of -1. Note that the run-lengths 63 and 64
(b111110 and b111111) are illegal as they are occupied by the QOI_OP_RGB and
QOI_OP_RGBA tags.


.- QOI_OP_RGB ------------------------------------------.
|         Byte[0]         | Byte[1] | Byte[2] | Byte[3] |
|  7  6  5  4  3  2  1  0 | 7 .. 0  | 7 .. 0  | 7 .. 0  |
|-------------------------+---------+---------+---------|
|  1  1  1  1  1  1  1  0 |   red   |  green  |  blue   |
`-------------------------------------------------------`
8-bit tag b11111110
8-bit   red channel value
8-bit green channel value
8-bit  blue channel value

The alpha value remains unchanged from the previous pixel.


.- QOI_OP_RGBA ---------------------------------------------------.
|         Byte[0]         | Byte[1] | Byte[2] | Byte[3] | Byte[4] |
|  7  6  5  4  3  2  1  0 | 7 .. 0  | 7 .. 0  | 7 .. 0  | 7 .. 0  |
|-------------------------+---------+---------+---------+---------|
|  1  1  1  1  1  1  1  1 |   red   |  green  |  blue   |  alpha  |
`-----------------------------------------------------------------`
8-bit tag b11111111
8-bit   red channel value
8-bit green channel value
8-bit  blue channel value
8-bit alpha channel value

*/
package main

const (
	QOI_SRGB   = 0
	QOI_LINEAR = 1
)

type qoi_desc struct {
	width      uint32
	height     uint32
	channels   byte
	colorspace byte
}

func QOI_ZEROARR(p [64]qoi_rgba_t) {

}

const (
	QOI_OP_INDEX byte = 0x00 /* 00xxxxxx */
	QOI_OP_DIFF  byte = 0x40 /* 01xxxxxx */
	QOI_OP_LUMA  byte = 0x80 /* 10xxxxxx */
	QOI_OP_RUN   byte = 0xc0 /* 11xxxxxx */
	QOI_OP_RGB   byte = 0xfe /* 11111110 */
	QOI_OP_RGBA  byte = 0xff /* 11111111 */

)

const QOI_MASK_2 = 0xc0 /* 11000000 */

func QOI_COLOR_HASH(C qoi_rgba_t) uint32 {
	return uint32(C.rgba.r*3 + C.rgba.g*5 + C.rgba.b*7 + C.rgba.a*11)
}

var QOI_MAGIC = [4]byte{0x71, 0x69, 0x6f, 0x66}

const QOI_HEADER_SIZE = 14

/* 2GB is the max file size that this implementation can safely handle. We guard
against anything larger than that, assuming the worst case with 5 bytes per
pixel, rounded down to a nice clean value. 400 million pixels ought to be
enough for anybody. */
const QOI_PIXELS_MAX = 400000000

type qoi_rgba_t struct {
	rgba rgba_t
	v    uint32 // ?
}

type rgba_t struct {
	r byte
	g byte
	b byte
	a byte
}

func bytes_to_rgba_t(bytes [4]byte) *rgba_t {
	var rgbaval rgba_t
	rgbaval.r = bytes[0]
	rgbaval.g = bytes[1]
	rgbaval.b = bytes[2]
	rgbaval.a = bytes[3]
	return &rgbaval
}

func fromBytes(bytes []byte) *qoi_rgba_t {
	var qrt qoi_rgba_t

	var b4 [4]byte
	for i := 0; i < 4; i++ {
		b4[i] = bytes[i]
	}

	qrt.v = bytes_to_uint32(b4)
	qrt.rgba.r = bytes[0]
	qrt.rgba.g = bytes[1]
	qrt.rgba.b = bytes[2]
	qrt.rgba.a = bytes[3]
	return &qrt
}

func fromUint32(val uint32) *qoi_rgba_t {
	var qrt qoi_rgba_t
	qrt.v = val
	bytes := uint32_to_bytes(val)
	qrt.rgba.r = bytes[0]
	qrt.rgba.g = bytes[1]
	qrt.rgba.b = bytes[2]
	qrt.rgba.a = bytes[3]
	return &qrt
}

var qoi_padding = [8]byte{0, 0, 0, 0, 0, 0, 0, 1}

func uint32_to_bytes(v uint32) []byte {
	bytes := [4]byte{0, 0, 0, 0}
	bytes[0] = byte((0xff000000 & v) >> 24)
	bytes[1] = byte((0x00ff0000 & v) >> 16)
	bytes[2] = byte((0x0000ff00 & v) >> 8)
	bytes[3] = byte(0x000000ff & v)
	return bytes[:]
}

func bytes_to_uint32(bytes [4]byte) uint32 {
	return uint32((bytes[0] << 24) + (bytes[1] << 16) + (bytes[2] << 8) + bytes[3])
}

// p means position
func qoi_write_32(bytes []byte, p *int, v uint32) {
	bytes[*p] = byte((0xff000000 & v) >> 24)
	(*p)++
	bytes[*p] = byte((0x00ff0000 & v) >> 16)
	(*p)++
	bytes[*p] = byte((0x0000ff00 & v) >> 8)
	(*p)++
	bytes[*p] = byte(0x000000ff & v)
	(*p)++
}

func qoi_read_32(bytes []byte, p *int) uint32 {
	a := bytes[*p]
	(*p)++
	b := bytes[*p]
	(*p)++
	c := bytes[*p]
	(*p)++
	d := bytes[*p]
	(*p)++

	return uint32((a << 24) + (b << 16) + (c << 8) + d)
}

func qoi_encode(data []byte, desc *qoi_desc, out_len *int) []byte {
	index := [64]qoi_rgba_t{}

	if (len(data) == 0) ||
		(out_len == nil) ||
		(desc == nil) || (desc.width == 0) || (desc.height == 0) ||
		(desc.channels < 3) || (desc.channels > 4) ||
		(desc.colorspace > 1) ||
		(desc.height >= QOI_PIXELS_MAX/desc.width) {
		return nil
	}

	max_size :=
		int(desc.width)*int(desc.height)*(int(desc.channels)+1) +
			QOI_HEADER_SIZE + len(qoi_padding)
	p := 0
	bytes := make([]byte, max_size)
	qoi_write_32(bytes, &p, bytes_to_uint32(QOI_MAGIC))
	qoi_write_32(bytes, &p, desc.width)
	qoi_write_32(bytes, &p, desc.height)
	bytes[p] = desc.channels
	p++
	bytes[p] = desc.colorspace
	p++

	pixels := data

	QOI_ZEROARR(index)

	var run byte = 0
	var px_prev qoi_rgba_t
	px_prev.rgba.r = 0
	px_prev.rgba.g = 0
	px_prev.rgba.b = 0
	px_prev.rgba.a = 255
	px := px_prev

	px_len := desc.width * desc.height * uint32(desc.channels)
	px_end := px_len - uint32(desc.channels)
	channels := uint32(desc.channels)

	var px_pos uint32
	for px_pos = 0; px_pos < px_len; px_pos += channels {
		if channels == 4 {
			px = *fromBytes(pixels[px_pos : px_pos+channels])
		} else {
			px.rgba.r = pixels[px_pos+0]
			px.rgba.g = pixels[px_pos+1]
			px.rgba.b = pixels[px_pos+2]
		}

		if px.v == px_prev.v {
			run++
			if run == 62 || px_pos == px_end {
				bytes[p] = byte(QOI_OP_RUN | (run - 1))
				p++
				run = 0
			}
		} else {
			if run > 0 {
				bytes[p] = byte(QOI_OP_RUN | (run - 1))
				p++
				run = 0
			}

			index_pos := byte(QOI_COLOR_HASH(px) % 64)

			if index[index_pos].v == px.v {
				bytes[p] = (QOI_OP_INDEX | index_pos)
				p++
			} else {
				index[index_pos] = px

				if px.rgba.a == px_prev.rgba.a {
					var vr int8 = int8(px.rgba.r - px_prev.rgba.r)
					var vg int8 = int8(px.rgba.g - px_prev.rgba.g)
					var vb int8 = int8(px.rgba.b - px_prev.rgba.b)

					var vg_r int8 = vr - vg
					var vg_b int8 = vb - vg

					if vr > -3 && vr < 2 &&
						vg > -3 && vg < 2 &&
						vb > -3 && vb < 2 {
						bytes[p] = QOI_OP_DIFF | byte((vr+2)<<4) | byte((vg+2)<<2) | byte(vb+2)
						p++
					} else if vg_r > -9 && vg_r < 8 &&
						vg > -33 && vg < 32 &&
						vg_b > -9 && vg_b < 8 {
						bytes[p] = QOI_OP_LUMA | byte(vg+32)
						p++
						bytes[p] = byte((vg_r+8)<<4) | byte(vg_b+8)
						p++
					} else {
						bytes[p] = QOI_OP_RGB
						p++
						bytes[p] = px.rgba.r
						p++
						bytes[p] = px.rgba.g
						p++
						bytes[p] = px.rgba.b
						p++
					}
				} else {
					bytes[p] = QOI_OP_RGBA
					p++
					bytes[p] = px.rgba.r
					p++
					bytes[p] = px.rgba.g
					p++
					bytes[p] = px.rgba.b
					p++
					bytes[p] = px.rgba.a
					p++
				}
			}
		}
		px_prev = px
	}

	for i := 0; i < len(qoi_padding); i++ {
		bytes[p] = qoi_padding[i]
		p++
	}

	*out_len = p
	return bytes
}
