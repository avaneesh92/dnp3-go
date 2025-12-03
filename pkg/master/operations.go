package master

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"avaneesh/dnp3-go/pkg/app"
	"avaneesh/dnp3-go/pkg/types"
)

// Scan operations

// AddIntegrityScan adds a periodic integrity scan
func (m *master) AddIntegrityScan(period time.Duration) (ScanHandle, error) {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	task := &IntegrityScanTask{
		id:       m.nextScanID,
		priority: PriorityNormal,
	}

	scan := &PeriodicScan{
		id:      m.nextScanID,
		task:    task,
		period:  period,
		nextRun: time.Now(),
		enabled: true,
	}

	m.scans[m.nextScanID] = scan
	m.nextScanID++

	// Schedule first run
	m.taskQueue.Push(task, task.Priority(), time.Now())

	m.logger.Info("Master %s: Added integrity scan (period=%s, id=%d)", m.config.ID, period, scan.id)

	return &ScanHandleImpl{id: scan.id, master: m}, nil
}

// AddClassScan adds a periodic class scan
func (m *master) AddClassScan(classes app.ClassField, period time.Duration) (ScanHandle, error) {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	task := &ClassScanTask{
		id:       m.nextScanID,
		classes:  classes,
		priority: PriorityNormal,
	}

	scan := &PeriodicScan{
		id:      m.nextScanID,
		task:    task,
		period:  period,
		nextRun: time.Now(),
		enabled: true,
	}

	m.scans[m.nextScanID] = scan
	m.nextScanID++

	// Schedule first run
	m.taskQueue.Push(task, task.Priority(), time.Now())

	m.logger.Info("Master %s: Added class scan %s (period=%s, id=%d)", m.config.ID, classes, period, scan.id)

	return &ScanHandleImpl{id: scan.id, master: m}, nil
}

// AddRangeScan adds a periodic range scan
func (m *master) AddRangeScan(objGroup, variation uint8, start, stop uint16, period time.Duration) (ScanHandle, error) {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	task := &RangeScanTask{
		id:        m.nextScanID,
		group:     objGroup,
		variation: variation,
		start:     start,
		stop:      stop,
		priority:  PriorityNormal,
	}

	scan := &PeriodicScan{
		id:      m.nextScanID,
		task:    task,
		period:  period,
		nextRun: time.Now(),
		enabled: true,
	}

	m.scans[m.nextScanID] = scan
	m.nextScanID++

	// Schedule first run
	m.taskQueue.Push(task, task.Priority(), time.Now())

	m.logger.Info("Master %s: Added range scan G%dV%d [%d-%d] (period=%s, id=%d)",
		m.config.ID, objGroup, variation, start, stop, period, scan.id)

	return &ScanHandleImpl{id: scan.id, master: m}, nil
}

// ScanIntegrity performs one-time integrity scan
func (m *master) ScanIntegrity() error {
	task := &IntegrityScanTask{
		id:       0,
		priority: PriorityHigh,
	}
	m.taskQueue.Push(task, task.Priority(), time.Now())
	return nil
}

// ScanClasses performs one-time class scan
func (m *master) ScanClasses(classes app.ClassField) error {
	task := &ClassScanTask{
		id:       0,
		classes:  classes,
		priority: PriorityHigh,
	}
	m.taskQueue.Push(task, task.Priority(), time.Now())
	return nil
}

// ScanRange performs one-time range scan
func (m *master) ScanRange(objGroup, variation uint8, start, stop uint16) error {
	task := &RangeScanTask{
		id:        0,
		group:     objGroup,
		variation: variation,
		start:     start,
		stop:      stop,
		priority:  PriorityHigh,
	}
	m.taskQueue.Push(task, task.Priority(), time.Now())
	return nil
}

// demandScan triggers an immediate scan
func (m *master) demandScan(id int) error {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	scan, exists := m.scans[id]
	if !exists {
		return errors.New("not implemented")
	}

	scan.demanded = true
	m.logger.Info("Master %s: Scan %d demanded", m.config.ID, id)
	return nil
}

// removeScan removes a periodic scan
func (m *master) removeScan(id int) error {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	delete(m.scans, id)
	m.logger.Info("Master %s: Scan %d removed", m.config.ID, id)
	return nil
}

// performIntegrityScan performs an integrity scan (Class 0)
func (m *master) performIntegrityScan() error {
	// Build READ request for Class 0
	objects := buildReadRequest(app.Class0)

	apdu := app.NewRequestAPDU(app.FuncRead, m.getNextSequence(), objects)

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performClassScan performs a class scan
func (m *master) performClassScan(classes app.ClassField) error {
	// Build READ request for classes
	objects := buildReadRequest(classes)

	apdu := app.NewRequestAPDU(app.FuncRead, m.getNextSequence(), objects)

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// performRangeScan performs a range scan
func (m *master) performRangeScan(group, variation uint8, start, stop uint16) error {
	// Build READ request for range
	var buf bytes.Buffer

	// Object header
	buf.WriteByte(group)
	buf.WriteByte(variation)
	buf.WriteByte(uint8(app.Qualifier16BitStartStop))

	// Range (start-stop, little-endian)
	binary.Write(&buf, binary.LittleEndian, start)
	binary.Write(&buf, binary.LittleEndian, stop)

	apdu := app.NewRequestAPDU(app.FuncRead, m.getNextSequence(), buf.Bytes())

	_, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	return err
}

// buildReadRequest builds a READ request for classes
func buildReadRequest(classes app.ClassField) []byte {
	var buf bytes.Buffer

	if classes&app.Class0 != 0 {
		// Class 0 (all static data)
		buf.WriteByte(app.GroupClass0Data)
		buf.WriteByte(app.VariationAny)
		buf.WriteByte(uint8(app.QualifierNoRange))
	}

	if classes&app.Class1 != 0 {
		// Class 1
		buf.WriteByte(app.GroupClass1Data)
		buf.WriteByte(app.VariationAny)
		buf.WriteByte(uint8(app.QualifierNoRange))
	}

	if classes&app.Class2 != 0 {
		// Class 2
		buf.WriteByte(app.GroupClass2Data)
		buf.WriteByte(app.VariationAny)
		buf.WriteByte(uint8(app.QualifierNoRange))
	}

	if classes&app.Class3 != 0 {
		// Class 3
		buf.WriteByte(app.GroupClass3Data)
		buf.WriteByte(app.VariationAny)
		buf.WriteByte(uint8(app.QualifierNoRange))
	}

	return buf.Bytes()
}

// Command operations

// SelectAndOperate performs SELECT then OPERATE
func (m *master) SelectAndOperate(commands []types.Command) ([]types.CommandStatus, error) {
	task := &CommandTask{
		commands:     commands,
		selectBefore: true,
		priority:     PriorityHigh,
		result:       make(chan CommandResult, 1),
	}

	m.taskQueue.Push(task, task.Priority(), time.Now())

	// Wait for result
	select {
	case result := <-task.result:
		return result.Statuses, result.Error
	case <-time.After(m.config.ResponseTimeout * 2): // SELECT + OPERATE
		return nil, ErrTimeout
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

// DirectOperate performs DIRECT OPERATE
func (m *master) DirectOperate(commands []types.Command) ([]types.CommandStatus, error) {
	task := &CommandTask{
		commands:     commands,
		selectBefore: false,
		priority:     PriorityHigh,
		result:       make(chan CommandResult, 1),
	}

	m.taskQueue.Push(task, task.Priority(), time.Now())

	// Wait for result
	select {
	case result := <-task.result:
		return result.Statuses, result.Error
	case <-time.After(m.config.ResponseTimeout):
		return nil, ErrTimeout
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

// performSelectAndOperate executes SELECT and OPERATE
func (m *master) performSelectAndOperate(commands []types.Command) ([]types.CommandStatus, error) {
	// SELECT phase
	selectAPDU := m.buildCommandAPDU(app.FuncSelect, commands)
	selectResp, err := m.sendAndWait(selectAPDU, m.config.ResponseTimeout)
	if err != nil {
		return nil, err
	}

	// Check SELECT response
	// TODO: Parse response and check status
	_ = selectResp // Use response

	// OPERATE phase
	operateAPDU := m.buildCommandAPDU(app.FuncOperate, commands)
	operateResp, err := m.sendAndWait(operateAPDU, m.config.ResponseTimeout)
	if err != nil {
		return nil, err
	}

	// Parse response and extract statuses
	// TODO: Full implementation
	statuses := make([]types.CommandStatus, len(commands))
	for i := range statuses {
		statuses[i] = types.CommandStatusSuccess
	}

	_ = operateResp // Use response
	return statuses, nil
}

// performDirectOperate executes DIRECT OPERATE
func (m *master) performDirectOperate(commands []types.Command) ([]types.CommandStatus, error) {
	apdu := m.buildCommandAPDU(app.FuncDirectOperate, commands)
	resp, err := m.sendAndWait(apdu, m.config.ResponseTimeout)
	if err != nil {
		return nil, err
	}

	// Parse response and extract statuses
	// TODO: Full implementation
	statuses := make([]types.CommandStatus, len(commands))
	for i := range statuses {
		statuses[i] = types.CommandStatusSuccess
	}

	_ = resp // Use response
	return statuses, nil
}

// buildCommandAPDU builds a command APDU
func (m *master) buildCommandAPDU(function app.FunctionCode, commands []types.Command) *app.APDU {
	// TODO: Build proper command request
	// For now, just create empty request
	return app.NewRequestAPDU(function, m.getNextSequence(), nil)
}
