package main

const DRIVER_HEADER_TEMPLATE =
`#pragma once

#ifndef NTSTRSAFE_LIB
#define NTSTRSAFE_LIB
#endif

extern "C"
{
#include <ntddk.h>
#include <wdm.h>
#include <ntstrsafe.h>
#include "..\Common\Common.h"
}

#ifndef _FN_
#define _FN_	__FUNCTION__
#endif

#ifndef _LN_
#define _LN_	__LINE__
#endif

#define DEVICE_NAME		L"\\Device\\$PROJECTNAME_SYS$"
#define DOS_DEVICE_NAME	L"\\DosDevices\\$PROJECTNAME_SYS$"

#define TAG_NAME        'TSET'


void
UnloadRoutine(
	IN	PDRIVER_OBJECT		pDriverObject
);

NTSTATUS
PassRoutine(
	IN	PDEVICE_OBJECT		pDeviceObject,
	IN	PIRP				pIrp
);

NTSTATUS
DeviceIoControlRoutine(
	IN	PDEVICE_OBJECT		pDeviceObject,
	IN	PIRP				pIrp
);
`

const DRIVER_CPP_TEMPLATE =
`#include "$PROJECTNAME_SYS$.h"


EXTERN_C
NTSTATUS
DriverEntry(
	IN	PDRIVER_OBJECT		pDriverObject,
	IN	PUNICODE_STRING		pRegistryPath
)
{
	UNREFERENCED_PARAMETER(pRegistryPath);

	KdPrint(("[%s:%d]\n", _FN_, _LN_));

	NTSTATUS status = STATUS_UNSUCCESSFUL;

	PDEVICE_OBJECT pDeviceObject = NULL;
	UNICODE_STRING symbolicLinkName = { 0 };

	do
	{		
		UNICODE_STRING deviceName;
		RtlInitUnicodeString(&deviceName, DEVICE_NAME);
		RtlInitUnicodeString(&symbolicLinkName, DOS_DEVICE_NAME);

		status = IoCreateSymbolicLink(&symbolicLinkName, &deviceName);
		if (!NT_SUCCESS(status))
		{
			break;
		}

		status = IoCreateDevice(
			pDriverObject,
			0,
			&deviceName,
			FILE_DEVICE_UNKNOWN,
			0,
			TRUE,
			&pDeviceObject
		);
		if (!NT_SUCCESS(status))
		{
			break;
		}

		for (int i = 0; i <= IRP_MJ_MAXIMUM_FUNCTION; ++i)
		{
			pDriverObject->MajorFunction[i] = PassRoutine;
		}

		pDriverObject->MajorFunction[IRP_MJ_DEVICE_CONTROL] = DeviceIoControlRoutine;
		pDriverObject->DriverUnload = UnloadRoutine;

		status = STATUS_SUCCESS;

	} while (false);

	if (!NT_SUCCESS(status))
	{
		if (symbolicLinkName.Buffer != NULL && symbolicLinkName.Length > 0)
		{
			IoDeleteSymbolicLink(&symbolicLinkName);
		}

		if (pDeviceObject != NULL)
		{
			IoDeleteDevice(pDeviceObject);
			pDeviceObject = NULL;
		}
	}

	return status;
}


void
UnloadRoutine(
	IN	PDRIVER_OBJECT		pDriverObject
)
{
	KdPrint(("[%s:%d]\n", _FN_, _LN_));

	UNICODE_STRING symbolicLinkName;
	RtlInitUnicodeString(&symbolicLinkName, DOS_DEVICE_NAME);
	IoDeleteSymbolicLink(&symbolicLinkName);

	if (pDriverObject->DeviceObject != NULL)
	{
		IoDeleteDevice(pDriverObject->DeviceObject);
		pDriverObject->DeviceObject = NULL;
	}

	return;
}


NTSTATUS
PassRoutine(
	IN	PDEVICE_OBJECT		pDeviceObject,
	IN	PIRP				pIrp
)
{
	UNREFERENCED_PARAMETER(pDeviceObject);

	pIrp->IoStatus.Status = STATUS_SUCCESS;
	pIrp->IoStatus.Information = 0;

	IoCompleteRequest(pIrp, IO_NO_INCREMENT);

	return STATUS_SUCCESS;
}


NTSTATUS
DeviceIoControlRoutine(
	IN	PDEVICE_OBJECT		pDeviceObject,
	IN	PIRP				pIrp
)
{
	UNREFERENCED_PARAMETER(pDeviceObject);

	NTSTATUS status = STATUS_UNSUCCESSFUL;
	ULONG_PTR information = 0;

	PIO_STACK_LOCATION pStack = IoGetCurrentIrpStackLocation(pIrp);
	ULONG ioCtlCode = pStack->Parameters.DeviceIoControl.IoControlCode;
	ULONG inSize = pStack->Parameters.DeviceIoControl.InputBufferLength;
	ULONG outSize = pStack->Parameters.DeviceIoControl.OutputBufferLength;
	PVOID pInBuffer = pIrp->AssociatedIrp.SystemBuffer;
	PVOID pOutBuffer = pIrp->AssociatedIrp.SystemBuffer;

	switch (ioCtlCode)
	{
	case IOCTL_MYDRIVER_1:
	{
		KdPrint(("[%s:%d] IOCTL_MYDRIVER_1. inSize : 0x%X\n", _FN_, _LN_, inSize));
		if (pInBuffer == NULL || inSize != sizeof(MYDRIVER_DATA_1))
		{
			status = STATUS_INVALID_PARAMETER;
			break;
		}

		status = STATUS_SUCCESS;
		break;
	}
	default:
		status = STATUS_INVALID_DEVICE_REQUEST;
		information = 0;
		break;
	}

	pIrp->IoStatus.Status = status;
	pIrp->IoStatus.Information = information;
	IoCompleteRequest(pIrp, IO_NO_INCREMENT);

	return status;
}
`

const EXE_CPP_TEMPLATE = 
`#include <iostream>
#include <Windows.h>
#include "../Common/Common.h"

int main()
{
	HANDLE deviceHandle = INVALID_HANDLE_VALUE;

	do
	{
		deviceHandle = CreateFile(L"\\\\.\\$PROJECTNAME_SYS$", GENERIC_READ | GENERIC_WRITE, 0, nullptr, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, nullptr);
		if (deviceHandle == INVALID_HANDLE_VALUE)
		{
			break;
		}

		DWORD retSize = 0;
		MYDRIVER_DATA_1 ioctlData = { 0 };
		ioctlData.totalSize = sizeof(ioctlData);

		if (!DeviceIoControl(deviceHandle, IOCTL_MYDRIVER_1, (LPVOID)&ioctlData, sizeof(ioctlData), nullptr, 0, &retSize, nullptr))
		{
			break;
		}

	} while (false);

	if (deviceHandle != INVALID_HANDLE_VALUE)
	{
		CloseHandle(deviceHandle);
		deviceHandle = INVALID_HANDLE_VALUE;
	}

	return 0;
}
`

const COMMON_HEADER_TEMPLATE = 
`#pragma once

#define IOCTL_MYDRIVER_1		CTL_CODE(FILE_DEVICE_UNKNOWN, 0x800, METHOD_BUFFERED, FILE_ANY_ACCESS)

#pragma pack (push, 1)

typedef struct _MYDRIVER_DATA_1
{
    ULONG               totalSize;
	   
} MYDRIVER_DATA_1, *PMYDRIVER_DATA_1;

#pragma pack (pop)
`