package dotlite

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	NodeIndexInt  = 0x02
	NodeTableInt  = 0x05
	NodeIndexLeaf = 0x0a
	NodeTableLeaf = 0x0d
)

// TreeHeader represents the header for a b-tree page in the sqlite database file
type TreeHeader struct {
	Kind            byte  // the type of the node
	FreeBlockOffset int16 // offset of the first freeblock on the page
	NumCells        int16 // number of cells on the page
	CellsOffset     int16 // offset into first byte of the cell content area
	NumFreeBytes    int8  // the number of fragmented free bytes within the cell content area.
}

// TreeNode represents an individual node in the tree
type TreeNode struct {
	file   *File      // reference to the database file
	header TreeHeader // header describing meta-information about this node
	page   *Page      // page backing this node
	cells  []int16    // offset of cells contained in this node

	// the right-most child pointer. This value appears in the header of interior b-tree pages only and is omitted from all other pages.
	right int32
}

// newNode parses a btree node from the given page
func newNode(file *File, page *Page) (_ *TreeNode, err error) {
	if page.ID == 1 {
		// skip first 100 bytes of the first page
		if _, err = page.Seek(100, io.SeekStart); err != nil {
			return nil, err
		}
	}

	var header TreeHeader
	if err = binary.Read(page, binary.BigEndian, &header); err != nil {
		return nil, err
	}

	var node = &TreeNode{file: file, header: header, page: page}
	if node.Kind() == NodeTableInt || node.Kind() == NodeIndexInt {
		if err = binary.Read(page, binary.BigEndian, &node.right); err != nil {
			return nil, err
		}
	}

	// TODO(@riyaz): using unsafe.Pointer can we directly map []int16 to the underlying page buffer?
	var cells = make([]int16, node.header.NumCells)
	for i := 0; i < len(cells); i++ {
		var cell int16
		if err = binary.Read(page, binary.BigEndian, &cell); err != nil {
			return nil, err
		}
		cells[i] = cell
	}

	node.cells = cells
	return node, nil
}

func (node *TreeNode) Kind() byte    { return node.header.Kind }
func (node *TreeNode) NumCells() int { return int(node.header.NumCells) }

// Cell is the data container for b-tree
type Cell struct {
	LeftChild int32 // page number of the left child
	Size      int64 // size of the byte payload (including overflow)
	Rowid     int64 // rowid of the row contained in this cell; valid only for b-tree holding tables

	data []byte // cell data buffer
}

func (cell *Cell) Reader() *bytes.Reader {
	return bytes.NewReader(cell.data)
}

func (node *TreeNode) LoadCell(pos int) (_ *Cell, err error) {
	var addr = int64(node.cells[pos])
	if _, err = node.page.Seek(addr, io.SeekStart); err != nil {
		return nil, err
	}

	switch k := node.Kind(); k {
	case NodeTableInt:
		var left int32
		if err = binary.Read(node.page, binary.BigEndian, &left); err != nil {
			return nil, err
		}

		var rowid int64
		if rowid, err = Varint(node.page); err != nil {
			return nil, fmt.Errorf("error decoding rowid: page=%d\tcell=%d", node.page.ID, pos)
		}

		return &Cell{LeftChild: left, Rowid: rowid}, nil

	case NodeTableLeaf:
		var size, rowid int64

		if size, err = Varint(node.page); err != nil {
			return nil, fmt.Errorf("error decoding size: page=%d\tcell=%d", node.page.ID, pos)
		}

		if rowid, err = Varint(node.page); err != nil {
			return nil, fmt.Errorf("error decoding rowid: page=%d\tcell=%d", node.page.ID, pos)
		}

		P := int(size)                                                              // the payload size
		U := int(node.file.Header.PageSize - uint16(node.file.Header.PageReserved)) // the usable page size of pages in the database
		X := U - 35                                                                 // maximum amount of payload that can be stored directly on the b-tree page

		// size of local (embedded in tree) and overflow content
		var localsz, overflowsz = P, 0

		// if the payload size > max embed value, then we calculate the amount of spillage
		if P > X {
			M := ((U - 12) * 32 / 255) - 23
			K := M + ((P - M) % (U - 4))

			localsz = K
			if K > X {
				localsz = M
			}

			overflowsz = P - localsz
		}

		// TODO(@riyaz): directly reference the underlying page buffer if there is no overflow?
		var payload = make([]byte, localsz)
		if _, err = io.ReadFull(node.page, payload); err != nil {
			return nil, err
		}

		if overflowsz != 0 {
			var overflowPage int32
			if err = binary.Read(node.page, binary.BigEndian, &overflowPage); err != nil {
				return nil, err
			}

			var usable = int(node.file.Header.PageSize - uint16(node.file.Header.PageReserved))
			var buffer = bytes.NewBuffer(payload)
			_, err = io.Copy(buffer, newOverflowReader(node.file.Pager, overflowPage, usable, overflowsz))
			if err != nil {
				return nil, err
			}

			payload = buffer.Bytes()
		}

		if len(payload) != P {
			return nil, fmt.Errorf("read %d payload bytes instead of %d", len(payload), P)
		}

		return &Cell{Size: int64(P), Rowid: rowid, data: payload}, err

	default:
		panic(fmt.Errorf("unknow node type: %v", k))
	}
}

// Tree represents a B-Tree in the sqlite database file
// see: https://www.sqlite.org/fileformat.html#b_tree_pages
type Tree struct {
	file  *File  // reference to the database file
	pager *Pager // pager used to fetch pages containing nodes of the tree
	root  int    // page containing the root node of the tree
}

// NewTree creates a new Tree using the provided pager, with page at r as the root
func NewTree(file *File, pager *Pager, root int) (_ *Tree) {
	return &Tree{file: file, pager: pager, root: root}
}

// Walk walks the tree using in-order traversal, invoking user-defined fn for each cell in all the nodes of the tree.
func (tree *Tree) Walk(fn func(*Cell) error) (err error) {
	var rootPage *Page
	if rootPage, err = tree.pager.ReadPage(tree.root); err != nil {
		return err
	}

	var root *TreeNode
	if root, err = newNode(tree.file, rootPage); err != nil {
		return err
	}

	return tree.walk(root, fn)
}

func (tree *Tree) walk(node *TreeNode, fn func(*Cell) error) (err error) {
	for i := 0; i < node.NumCells(); i++ {
		var cell *Cell
		if cell, err = node.LoadCell(i); err != nil {
			return err
		}

		if cell.LeftChild != 0 {
			var page *Page
			if page, err = tree.pager.ReadPage(int(cell.LeftChild)); err != nil {
				return err
			}

			var child *TreeNode
			if child, err = newNode(tree.file, page); err != nil {
				return err
			}

			if err = tree.walk(child, fn); err != nil {
				return err
			}
		}

		if node.Kind() != NodeTableInt {
			if err = fn(cell); err != nil {
				return err
			}
		}
	}

	if node.right != 0 {
		var page *Page
		if page, err = tree.pager.ReadPage(int(node.right)); err != nil {
			return err
		}

		var child *TreeNode
		if child, err = newNode(tree.file, page); err != nil {
			return err
		}

		if err = tree.walk(child, fn); err != nil {
			return err
		}
	}

	return nil
}
