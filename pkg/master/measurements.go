package master

import (
	"avaneesh/dnp3-go/pkg/app"
	"avaneesh/dnp3-go/pkg/types"
)

// processMeasurements processes measurement data from response
func (m *master) processMeasurements(apdu *app.APDU) {
	info := ResponseInfo{
		Unsolicited: apdu.FunctionCode == app.FuncUnsolicitedResponse,
		FIR:         apdu.FIR,
		FIN:         apdu.FIN,
	}

	m.callbacks.OnBeginFragment(info)

	// Parse object headers
	parser := app.NewParser(apdu.Objects)

	for parser.HasMore() {
		header, err := parser.ReadObjectHeader()
		if err != nil {
			m.logger.Error("Master %s: Failed to parse object header: %v", m.config.ID, err)
			break
		}

		headerInfo := HeaderInfo{
			Group:     header.Group,
			Variation: header.Variation,
			Qualifier: uint8(header.Qualifier),
			IsEvent:   isEventGroup(header.Group),
		}

		// Process based on group
		switch header.Group {
		case app.GroupBinaryInput, app.GroupBinaryInputEvent:
			m.processBinaryObjects(parser, header, headerInfo)
		case app.GroupAnalogInput, app.GroupAnalogInputEvent:
			m.processAnalogObjects(parser, header, headerInfo)
		case app.GroupCounter, app.GroupCounterEvent:
			m.processCounterObjects(parser, header, headerInfo)
		default:
			// Skip unknown group
			count := app.GetCount(header.Range)
			objectSize := getObjectSize(header.Group, header.Variation)
			if objectSize > 0 {
				parser.Skip(int(count) * objectSize)
			}
		}
	}

	m.callbacks.OnEndFragment(info)
}

// processBinaryObjects processes binary input objects
func (m *master) processBinaryObjects(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	// TODO: Parse based on variation
	// For now, just skip
	count := app.GetCount(header.Range)
	objectSize := getObjectSize(header.Group, header.Variation)
	if objectSize > 0 {
		parser.Skip(int(count) * objectSize)
	}

	// Placeholder - would parse actual values
	values := make([]types.IndexedBinary, 0)
	m.callbacks.ProcessBinary(info, values)
}

// processAnalogObjects processes analog input objects
func (m *master) processAnalogObjects(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	// TODO: Parse based on variation
	// For now, just skip
	count := app.GetCount(header.Range)
	objectSize := getObjectSize(header.Group, header.Variation)
	if objectSize > 0 {
		parser.Skip(int(count) * objectSize)
	}

	// Placeholder - would parse actual values
	values := make([]types.IndexedAnalog, 0)
	m.callbacks.ProcessAnalog(info, values)
}

// processCounterObjects processes counter objects
func (m *master) processCounterObjects(parser *app.Parser, header *app.ObjectHeader, info HeaderInfo) {
	// TODO: Parse based on variation
	// For now, just skip
	count := app.GetCount(header.Range)
	objectSize := getObjectSize(header.Group, header.Variation)
	if objectSize > 0 {
		parser.Skip(int(count) * objectSize)
	}

	// Placeholder - would parse actual values
	values := make([]types.IndexedCounter, 0)
	m.callbacks.ProcessCounter(info, values)
}

// isEventGroup returns true if the group is an event group
func isEventGroup(group uint8) bool {
	switch group {
	case app.GroupBinaryInputEvent,
		app.GroupDoubleBitBinaryEvent,
		app.GroupCounterEvent,
		app.GroupFrozenCounterEvent,
		app.GroupAnalogInputEvent,
		app.GroupFrozenAnalogEvent,
		app.GroupBinaryOutputEvent,
		app.GroupAnalogOutputEvent:
		return true
	default:
		return false
	}
}

// getObjectSize returns the size of an object in bytes
// Returns 0 for variable-size objects
func getObjectSize(group, variation uint8) int {
	// Simplified - actual sizes depend on variation
	switch group {
	case app.GroupBinaryInput:
		if variation == app.BinaryInputWithFlags {
			return 1 // 1 byte flags
		}
	case app.GroupAnalogInput:
		switch variation {
		case app.AnalogInput16Bit:
			return 3 // 2 bytes value + 1 byte flags
		case app.AnalogInput32Bit:
			return 5 // 4 bytes value + 1 byte flags
		case app.AnalogInputFloat:
			return 5 // 4 bytes float + 1 byte flags
		case app.AnalogInputDouble:
			return 9 // 8 bytes double + 1 byte flags
		}
	case app.GroupCounter:
		switch variation {
		case app.Counter16Bit:
			return 2
		case app.Counter32Bit:
			return 4
		case app.Counter16BitWithFlag:
			return 3
		case app.Counter32BitWithFlag:
			return 5
		}
	}
	return 0
}
