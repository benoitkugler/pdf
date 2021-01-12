package bitmap

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// parser for .pcf bitmap fonts

// ported from https://github.com/stumpycr/pcf-parser
// https://fontforge.org/docs/techref/pcf-format.html

const (
	properties = 1 << iota
	accelerators
	metrics
	bitmaps
	inkMetrics
	bdfEncodings
	sWidths
	glyphNames
	bdfAccelerators
)

const (
	PCF_DEFAULT_FORMAT     = 0x00000000
	PCF_INKBOUNDS          = 0x00000200
	PCF_ACCEL_W_INKBOUNDS  = 0x00000100
	PCF_COMPRESSED_METRICS = 0x00000100

	// modifiers
	PCF_GLYPH_PAD_MASK = 3 << 0 /* See the bitmap table for explanation */
	PCF_BYTE_MASK      = 1 << 2 /* If set then Most Sig Byte First */
	PCF_BIT_MASK       = 1 << 3 /* If set then Most Sig Bit First */
	PCF_SCAN_UNIT_MASK = 3 << 4 /* See the bitmap table for explanation */

	formatMask = ^uint32(0xFF) // keep the higher bits
)

const HEADER = "\x01fcp"

type Font struct {
	properties          propertiesTable
	bitmap              bitmapTable
	metrics, inkMetrics metricsTable
	encoding            encodingTable
}

type tocEntry struct {
	kind, format, size, offset uint32
}

type prop struct {
	nameOffset   uint32
	isStringProp bool
	value        uint32
}

type propertiesTable struct {
	props   []prop
	rawData []byte
}

type bitmapTable struct {
	data []byte
}

// we use int16 even for compressed for simplicity
type metric struct {
	leftSidedBearing    int16
	rightSidedBearing   int16
	characterWidth      int16
	characterAscent     int16
	characterDescent    int16
	characterAttributes uint16
}

type metricsTable []metric

type encodingTable map[uint16]uint16

func getOrder(format uint32) binary.ByteOrder {
	if format&PCF_BYTE_MASK != 0 {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

type parser struct {
	data []byte
	pos  int
}

func (p *parser) u32(order binary.ByteOrder) (uint32, error) {
	if len(p.data) < p.pos+4 {
		return 0, errors.New("corrupted font file")
	}
	out := order.Uint32(p.data[p.pos:])
	p.pos += 4
	return out, nil
}

func (p *parser) u16(order binary.ByteOrder) (uint16, error) {
	if len(p.data) < p.pos+2 {
		return 0, errors.New("corrupted font file")
	}
	out := order.Uint16(p.data[p.pos:])
	p.pos += 2
	return out, nil
}

func (p *parser) tocEntry() (out tocEntry, err error) {
	if len(p.data) < p.pos+16 {
		return out, errors.New("corrupted toc entry")
	}
	out.kind = binary.LittleEndian.Uint32(p.data[p.pos:])
	out.format = binary.LittleEndian.Uint32(p.data[p.pos+4:])
	out.size = binary.LittleEndian.Uint32(p.data[p.pos+8:])
	out.offset = binary.LittleEndian.Uint32(p.data[p.pos+12:])
	p.pos += 16
	return out, nil
}

const propSize = 9

func (pr *parser) prop(order binary.ByteOrder) (prop, error) {
	if len(pr.data) < pr.pos+propSize {
		return prop{}, errors.New("invalid property")
	}
	var out prop
	out.nameOffset = order.Uint32(pr.data[pr.pos:])
	out.isStringProp = pr.data[pr.pos+4] == 1
	out.value = order.Uint32(pr.data[pr.pos+5:])
	pr.pos += propSize
	return out, nil
}

func (pr *parser) propertiesTable() (propertiesTable, error) {
	format, err := pr.u32(binary.LittleEndian)
	if err != nil {
		return propertiesTable{}, err
	}

	order := getOrder(format)

	nprops, err := pr.u32(order)
	if err != nil {
		return propertiesTable{}, err
	}
	var out propertiesTable

	if len(pr.data) < pr.pos+int(nprops)*propSize {
		return propertiesTable{}, errors.New("invalid properties table")
	}
	out.props = make([]prop, nprops)
	for i := range out.props {
		out.props[i], err = pr.prop(order)
		if err != nil {
			return propertiesTable{}, err
		}
	}

	if padding := int(nprops & 3); padding != 0 {
		pr.pos += 4 - padding // padding
	}

	stringsLength, err := pr.u32(order)
	if err != nil {
		return propertiesTable{}, err
	}

	if len(pr.data) < pr.pos+int(stringsLength) {
		return propertiesTable{}, errors.New("invalid properties table")
	}

	out.rawData = pr.data[pr.pos : pr.pos+int(stringsLength)]
	return out, nil
}

//   class PropertiesTable
//     getter properties : Hash(String, (String | Int32))

//     def initialize(io)
//       @properties = {} of String => (String | Int32)

//       format = io.read_bytes(Int32, IO::ByteFormat::LittleEndian)
//       byte_mask = (format & 4) != 0 # set => most significant byte first
//       bit_mask = (format & 8) != 0  # set => most significant bit first

//       unless bit_mask
//         puts "Unsupported bit_mask: #{bit_mask}"
//       end

//       byte_format = byte_mask ? IO::ByteFormat::BigEndian : IO::ByteFormat::BigEndian

//       # :compressed_metrics is equiv. to :accel_w_inkbounds
//       main_format = [:default, :inkbounds, :compressed_metrics][format >> 8]

//       size = io.read_bytes(Int32, byte_format)
//       props = [] of Prop
//       size.times do
//         props << Prop.new(io, byte_format)
//       end

//       padding = (size & 3) == 0 ? 0 : 4 - (size & 3)
//       io.skip(padding)

//       string_size = io.read_bytes(Int32, byte_format)

//       # Start of the strings array
//       strings = io.pos
//       props.each do |prop|
//         name = nil
//         io.seek(strings + prop.name_offset) do
//           name = io.gets('\0', true)
//         end

//         raise "Could not read property name" if name.nil?

//         offset = prop.value
//         if prop.is_string_prop
//           io.seek(strings + offset) do
//             value = io.gets('\0', true)
//             raise "Could not read property value" if value.nil?
//             @properties[name] = value
//           end
//         else
//           @properties[name] = offset
//         end
//       end
//     end
//   end

func (p *parser) bitmap() (bitmapTable, error) {
	format, err := p.u32(binary.LittleEndian)
	if err != nil {
		return bitmapTable{}, err
	}
	if format&formatMask != PCF_DEFAULT_FORMAT {
		return bitmapTable{}, fmt.Errorf("invalid bitmap format: %d", format)
	}

	order := getOrder(format)

	count, err := p.u32(order)
	if err != nil {
		return bitmapTable{}, err
	}

	if len(p.data) < p.pos+int(count)*4 {
		return bitmapTable{}, fmt.Errorf("invalid bitmap table")
	}
	offsets := make([]uint32, count)
	for i := range offsets {
		offsets[i] = order.Uint32(p.data[p.pos+i*4:])
	}
	p.pos += int(count) * 4

	var sizes [4]uint32
	if len(p.data) < p.pos+16 {
		return bitmapTable{}, fmt.Errorf("invalid bitmap table")
	}
	sizes[0] = order.Uint32(p.data[p.pos:])
	sizes[1] = order.Uint32(p.data[p.pos+4:])
	sizes[2] = order.Uint32(p.data[p.pos+8:])
	sizes[3] = order.Uint32(p.data[p.pos+12:])
	p.pos += 16

	bitmapLength := int(sizes[format&3])
	if len(p.data) < p.pos+bitmapLength {
		return bitmapTable{}, fmt.Errorf("invalid bitmap table")
	}
	data := p.data[p.pos : p.pos+bitmapLength]
	p.pos += bitmapLength

	return bitmapTable{data: data}, nil
}

//   class BitmapTable
//     getter bitmaps : Array(Bytes)
//     getter padding_bytes : Int32
//     getter data_bytes : Int32

//     # TODO: Raise if format != PCF_DEFAULT
//     def initialize(io)
//       format = io.read_bytes(Int32, IO::ByteFormat::LittleEndian)

//       # 0 => byte (8bit), 1 => short (16bit), 2 => int (32bit)
//       # TODO: What is this needed for?
//       glyph_pad = format & 3
//       @padding_bytes = glyph_pad == 0 ? 1 : glyph_pad * 2

//       byte_mask = (format & 4) != 0 # set => most significant byte first
//       bit_mask = (format & 8) != 0  # set => most significant bit first

//       # 0 => byte (8bit), 1 => short (16bit), 2 => int (32bit)
//       scan_unit = (format >> 4) & 3
//       @data_bytes = scan_unit == 0 ? 1 : scan_unit * 2

//       puts "Unsupported bit_mask: #{bit_mask}" unless bit_mask
//       byte_format = byte_mask ? IO::ByteFormat::BigEndian : IO::ByteFormat::BigEndian

//       # :compressed_metrics is equiv. to :accel_w_inkbounds
//       main_format = [:default, :inkbounds, :compressed_metrics][format >> 8]

//       glyph_count = io.read_bytes(Int32, byte_format)
//       offsets = [] of Int32

//       glyph_count.times do
//         offsets << io.read_bytes(Int32, byte_format)
//       end

//       bitmap_sizes = [] of Int32
//       4.times do
//         bitmap_sizes << io.read_bytes(Int32, byte_format)
//       end

//       @bitmaps = [] of Bytes

//       slice = Bytes.new(bitmap_sizes[glyph_pad])
//       read = io.read_fully(slice)

//       raise "Failed to read bitmap data" if bitmap_sizes[glyph_pad] != read

//       offsets.each do |off|
//         @bitmaps << (slice + off)
//       end

//       # bitmap_data = io.pos
//       # offsets.each do |off|
//       #   size = bitmap_sizes[glyph_pad] / glyph_count
//       #   slice = Bytes.new(size)

//       #   io.seek(bitmap_data + off) do
//       #     n_read = io.read(slice)
//       #     raise "Failed to read bitmap data" if n_read != size
//       #     @bitmaps << slice
//       #   end
//       # end
//     end
//   end

func (pr *parser) metric(compressed bool, order binary.ByteOrder) (metric, error) {
	var out metric
	if compressed {
		if len(pr.data) < pr.pos+5 {
			return out, fmt.Errorf("invalid compressed metric data")
		}
		out.leftSidedBearing = int16(pr.data[pr.pos] - 0x80)
		out.rightSidedBearing = int16(pr.data[pr.pos+1] - 0x80)
		out.characterWidth = int16(pr.data[pr.pos+2] - 0x80)
		out.characterAscent = int16(pr.data[pr.pos+3] - 0x80)
		out.characterDescent = int16(pr.data[pr.pos+4] - 0x80)
		pr.pos += 5
	} else {
		if len(pr.data) < pr.pos+12 {
			return out, fmt.Errorf("invalid uncompressed metric data")
		}
		out.leftSidedBearing = int16(order.Uint16(pr.data[pr.pos:]))
		out.rightSidedBearing = int16(order.Uint16(pr.data[pr.pos+2:]))
		out.characterWidth = int16(order.Uint16(pr.data[pr.pos+4:]))
		out.characterAscent = int16(order.Uint16(pr.data[pr.pos+6:]))
		out.characterDescent = int16(order.Uint16(pr.data[pr.pos+8:]))
		out.characterAttributes = order.Uint16(pr.data[pr.pos+10:])
		pr.pos += 12
	}
	return out, nil
}

func (pr *parser) metricTable() (metricsTable, error) {
	format, err := pr.u32(binary.LittleEndian)
	if err != nil {
		return nil, err
	}

	order := getOrder(format)

	compressed := format&formatMask == PCF_COMPRESSED_METRICS&formatMask
	var count int
	if compressed {
		c, er := pr.u16(order)
		count, err = int(c), er
	} else {
		c, er := pr.u32(order)
		count, err = int(c), er
	}
	if err != nil {
		return nil, err
	}

	out := make(metricsTable, count)
	for i := range out {
		out[i], err = pr.metric(compressed, order)
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (pr *parser) encodingTable() (encodingTable, error) {
	format, err := pr.u32(binary.LittleEndian)
	if err != nil {
		return nil, err
	}

	if format&formatMask != PCF_DEFAULT_FORMAT {
		return nil, fmt.Errorf("invalid encoding table format: %d", format)
	}

	order := getOrder(format)

	if len(pr.data) < pr.pos+10 {
		return nil, fmt.Errorf("invalid encoding table")
	}

	minChar := order.Uint16(pr.data[pr.pos:])
	maxChar := order.Uint16(pr.data[pr.pos+2:])
	minByte := order.Uint16(pr.data[pr.pos+4:])
	maxByte := order.Uint16(pr.data[pr.pos+6:])
	defaultChar := order.Uint16(pr.data[pr.pos+8:])
	pr.pos += 10

	count := int(maxByte-minByte+1) * int(maxChar-minChar+1)
	if len(pr.data) < pr.pos+2*count {
		return nil, fmt.Errorf("invalid encoding table")

	}
	out := make(encodingTable, count)

	for ma := minByte; ma <= maxByte; ma++ {
		for mi := minChar; mi <= maxChar; mi++ {
			value := order.Uint16(pr.data[pr.pos:])
			pr.pos += 2

			full := mi | ma<<8
			if value != 0xffff {
				out[full] = value
			} else {
				out[full] = defaultChar
			}
		}
	}

	return out, nil
}

func parse(data []byte) (*Font, error) {
	if len(data) < 4 || string(data[0:4]) != HEADER {
		return nil, errors.New("not a PCF file")
	}

	pr := parser{data: data, pos: 4}
	tableCount, err := pr.u32(binary.LittleEndian)
	if err != nil {
		return nil, err
	}
	tocEntries := make([]tocEntry, tableCount)
	for i := range tocEntries {
		tocEntries[i], err = pr.tocEntry()
		if err != nil {
			return nil, err
		}
	}
	fmt.Println(tocEntries)
	var out Font
	for _, tc := range tocEntries {
		pr.pos = int(tc.offset) // seek
		switch tc.kind {
		case properties:
			out.properties, err = pr.propertiesTable()
		case bitmaps:
			out.bitmap, err = pr.bitmap()
		case metrics:
			out.metrics, err = pr.metricTable()
		case inkMetrics:
			out.inkMetrics, err = pr.metricTable()
		case bdfEncodings:
			out.encoding, err = pr.encodingTable()
		}
		if err != nil {
			return nil, err
		}
	}

	return &out, nil
}

//       bitmap_table = nil
//       metrics_table = nil
//       encoding_table = nil

//       @encoding = {} of Int16 => Int16

//       tocEntries.each do |entry|
//         io.seek(entry.offset)

//         case entry.type
//           # when TableType::properties
//           #   @properties_table = PropertiesTable.new(io)
//           when TableType::bitmaps
//             bitmap_table = BitmapTable.new(io)
//           when TableType::metrics
//             metrics_table = MetricsTable.new(io)
//           when TableType::bdfEncodings
//             encoding_table = EncodingTable.new(io)
//         end
//       end

//       raise "Could not find a bitmap table" if bitmap_table.nil?
//       raise "Could not find a metrics table" if metrics_table.nil?

//       bitmaps = bitmap_table.bitmaps
//       metrics = metrics_table.metrics

//       if bitmaps.size != metrics.size
//         raise "Bitmap and metrics tables are not of the same size"
//       end

//       unless encoding_table.nil?
//         @encoding = encoding_table.encoding
//       end

//       @characters = [] of Character

//       @max_ascent = 0
//       @max_descent = 0

//       bitmaps.each_with_index do |bitmap, i|
//         metric = metrics[i]

//         if metric.characterAscent > @max_ascent
//           @max_ascent += metric.characterAscent
//         end

//         if metric.characterDescent > @max_descent
//           @max_descent += metric.characterDescent
//         end

//         char = Character.new(
//           bitmap,
//           metric.characterWidth,
//           metric.characterAscent,
//           metric.characterDescent,
//           metric.leftSidedBearing,
//           metric.rightSidedBearing,
//           bitmap_table.data_bytes,
//           bitmap_table.padding_bytes,
//         )

//         @characters << char
//       end
//     end

//     def lookup(str : String)
//       str.chars.map { |c| lookup(c) }
//     end

//     def lookup(char : Char)
//       lookup(char.ord)
//     end

//     def lookup(char)
//       @characters[@encoding[char]]
//     end
//   end

//   class Character
//     getter width : Int16
//     getter ascent : Int16
//     getter descent : Int16

//     getter leftSidedBearing : Int16
//     getter rightSidedBearing : Int16

//     @padding_bytes : Int32
//     @data_bytes : Int32
//     @bytes : Bytes

//     @bytes_per_row : Int32

//     def initialize(@bytes, @width, @ascent, @descent, @leftSidedBearing, @rightSidedBearing, @data_bytes, @padding_bytes)
//       @bytes_per_row = [(@width / 8).to_i32, 1].max

//       # Pad as needed
//       if (@bytes_per_row % @padding_bytes) != 0
//         @bytes_per_row += @padding_bytes - (@bytes_per_row % @padding_bytes)
//       end

//       # TODO: Is this last row relevant?
//       @bytes_per_row = [@bytes_per_row, @data_bytes].max

//       # needed = @bytes_per_row * (@ascent + @descent)
//       # got = @bytes.size
//     end

//     def get(x, y)
//       unless 0 <= x < @width
//         raise "Invalid x value: #{x}, must be in range (0..#{@width})"
//       end

//       unless 0 <= y < (@ascent + @descent)
//         raise "Invalid y value: #{y}, must be in range (0..#{@ascent + @descent})"
//       end

//       index = x // 8 + @bytes_per_row * y
//       shift = 7 - (x % 8)

//       if index < @bytes.size
//         @bytes[index] & (1 << (7 - (x % 8))) != 0
//       else
//         true
//       end
//     end
//   end

//   class tocEntry
//     getter type : TableType
//     getter format : Int32
//     getter size : Int32
//     getter offset : Int32

//     def initialize(io)
//       @type = TableType.new(io.read_bytes(Int32, IO::ByteFormat::LittleEndian))
//       @format = io.read_bytes(Int32, IO::ByteFormat::LittleEndian)
//       @size   = io.read_bytes(Int32, IO::ByteFormat::LittleEndian)
//       @offset = io.read_bytes(Int32, IO::ByteFormat::LittleEndian)
//     end
//   end

//   class Prop
//     getter name_offset : Int32
//     getter is_string_prop : Bool
//     getter value : Int32

//     def initialize(io, byte_format)
//       @name_offset = io.read_bytes(Int32, byte_format)
//       @is_string_prop = io.read_bytes(Int8, byte_format) == 1
//       @value = io.read_bytes(Int32, byte_format)
//     end
//   end

//   class PropertiesTable
//     getter properties : Hash(String, (String | Int32))

//     def initialize(io)
//       @properties = {} of String => (String | Int32)

//       format = io.read_bytes(Int32, IO::ByteFormat::LittleEndian)
//       byte_mask = (format & 4) != 0 # set => most significant byte first
//       bit_mask = (format & 8) != 0  # set => most significant bit first

//       unless bit_mask
//         puts "Unsupported bit_mask: #{bit_mask}"
//       end

//       byte_format = byte_mask ? IO::ByteFormat::BigEndian : IO::ByteFormat::BigEndian

//       # :compressed_metrics is equiv. to :accel_w_inkbounds
//       main_format = [:default, :inkbounds, :compressed_metrics][format >> 8]

//       size = io.read_bytes(Int32, byte_format)
//       props = [] of Prop
//       size.times do
//         props << Prop.new(io, byte_format)
//       end

//       padding = (size & 3) == 0 ? 0 : 4 - (size & 3)
//       io.skip(padding)

//       string_size = io.read_bytes(Int32, byte_format)

//       # Start of the strings array
//       strings = io.pos
//       props.each do |prop|
//         name = nil
//         io.seek(strings + prop.name_offset) do
//           name = io.gets('\0', true)
//         end

//         raise "Could not read property name" if name.nil?

//         offset = prop.value
//         if prop.is_string_prop
//           io.seek(strings + offset) do
//             value = io.gets('\0', true)
//             raise "Could not read property value" if value.nil?
//             @properties[name] = value
//           end
//         else
//           @properties[name] = offset
//         end
//       end
//     end
//   end
// end
